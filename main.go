/*************************************************************************
 *
 * Copyright 2018 Ideas2IT Technology Services Private Limited.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 ***********************************************************************/

package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	_ "net/http/pprof"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/fetchbot"
	"github.com/PuerkitoBio/goquery"
	"github.com/PuerkitoBio/purell"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	pb "github.com/ideas2it/ideacrawler/protofiles"
	sc "github.com/ideas2it/ideacrawler/statuscodes"
	"github.com/shsms-i2i/sflag"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// TODO - implement the DEBUG feature
var cliParams = struct {
	DialAddress    string "Interface to dial from.  Defaults to OS defined routes|"
	SaveLoginPages string "Saves login pages in this path. Don't save them if empty."
	ListenAddress  string "Interface to listen on. Defaults to 127.0.0.1|127.0.0.1"
	ListenPort     string "Port to listen on. Defaults to 2345|2345"
	LogPath        string "Logpath to log into. Default is stdout."
}{}

type subscriber struct {
	doneSeqnum           int32
	reqChan              chan pb.PageRequest
	sendChan             chan pb.PageHTML
	stopChan             chan bool
	analyzedURLChan      chan pb.UrlList
	stopAnalyzedURLChan  chan bool
	analyzedURLConnected bool
	connected            bool
}

type cancelSignal struct{}
type jobDoneSignal struct{}

type ideaCrawlerServer struct {
	jobs       map[string]*job
	newJobChan chan<- newJob
	newSubChan chan<- newSub
}

type newJobStatus struct {
	sub                  pb.Subscription
	subscriber           *subscriber
	cancelChan           chan cancelSignal
	registerDoneListener chan chan jobDoneSignal
	err                  error
}

type newJob struct {
	opts      *pb.DomainOpt
	retChan   chan<- newJobStatus
	subscribe bool
}

type newSub struct {
	sub     pb.Subscription
	retChan chan<- newJobStatus
}

type CrawlCommand struct {
	method     string
	url        *url.URL
	noCallback bool
	metaStr    string
	urlDepth   int32
	anchorText string
}

func newCrawlCommand(method, urlstr, metaStr string, urlDepth int32) (CrawlCommand, error) {
	parsed, err := url.Parse(urlstr)
	return CrawlCommand{
		method:   method,
		url:      parsed,
		metaStr:  metaStr,
		urlDepth: urlDepth,
	}, err
}

func (c CrawlCommand) URL() *url.URL {
	return c.url
}

func (c CrawlCommand) Method() string {
	return c.method
}

func (c CrawlCommand) MetaStr() string {
	return c.metaStr
}

func (c CrawlCommand) URLDepth() int32 {
	return c.urlDepth
}

func domainNameFromURL(_url string) (string, error) { //returns domain name and error if any
	u, err := url.Parse(_url)
	if err != nil {
		return "", err
	}
	return u.Hostname(), nil
}

// addNewJob does validation on incoming jobs from clients, and adds
// them to the server's job pool.
func (s *ideaCrawlerServer) addNewJob(nj newJob) {
	log.Println("Received new job", nj.opts.SeedUrl)
	domainname, err := domainNameFromURL(nj.opts.SeedUrl)
	var jobStatusFailureMessage = func(err error) newJobStatus {
		return newJobStatus{
			sub:                  pb.Subscription{},
			subscriber:           &subscriber{},
			cancelChan:           nil,
			registerDoneListener: nil,
			err:                  err,
		}
	}
	if err != nil {
		nj.retChan <- jobStatusFailureMessage(err)
		return
	}
	emptyTS, _ := ptypes.TimestampProto(time.Time{})
	sub := pb.Subscription{
		Subcode:    uuid.New().String(),
		Domainname: domainname,
		Subtype:    pb.SubType_SEQNUM,
		Seqnum:     0,
		Datetime:   emptyTS,
	}

	freq, err := ptypes.Duration(nj.opts.Frequency)
	if nj.opts.Repeat == true && err != nil {
		nj.retChan <- jobStatusFailureMessage(fmt.Errorf("Bad value for DomainOpt.Frequency field - %s - %s", domainname, err))
		return
	}
	subr := &subscriber{}
	if nj.subscribe == true {
		subr = &subscriber{
			doneSeqnum:           0,
			reqChan:              make(chan pb.PageRequest, 1000),
			sendChan:             make(chan pb.PageHTML, 1000),
			stopChan:             make(chan bool, 3),
			analyzedURLChan:      nil,
			stopAnalyzedURLChan:  nil,
			analyzedURLConnected: false,
			connected:            true,
		}
	}
	var callbackURLRegexp, followURLRegexp, anchorTextRegexp *regexp.Regexp
	if len(nj.opts.CallbackUrlRegexp) > 0 {
		callbackURLRegexp, err = regexp.Compile(nj.opts.CallbackUrlRegexp)
		if err != nil {
			nj.retChan <- jobStatusFailureMessage(fmt.Errorf("CallbackUrlRegexp doesn't compile - %s - %s'", nj.opts.CallbackUrlRegexp, err))
			return
		}
	}
	if len(nj.opts.FollowUrlRegexp) > 0 {
		followURLRegexp, err = regexp.Compile(nj.opts.FollowUrlRegexp)
		if err != nil {
			nj.retChan <- jobStatusFailureMessage(fmt.Errorf("FollowURLRegexp doesn't compile - %s - %s", nj.opts.FollowUrlRegexp, err))
			return
		}
	}
	if len(nj.opts.AnchorTextRegexp) > 0 {
		anchorTextRegexp, err = regexp.Compile(nj.opts.AnchorTextRegexp)
		if err != nil {
			nj.retChan <- jobStatusFailureMessage(fmt.Errorf("AnchorTextRegexp doesn't compile - %s - %s", nj.opts.AnchorTextRegexp, err))
			return
		}
	}
	canc := make(chan cancelSignal)
	regDoneC := make(chan chan jobDoneSignal)
	randChan := make(chan int, 5)
	s.jobs[sub.Subcode] = &job{
		domainname:           domainname,
		opts:                 nj.opts,
		sub:                  sub,
		prevRun:              time.Time{},
		nextRun:              time.Time{},
		frequency:            freq,
		runNumber:            0,
		running:              false,
		done:                 false,
		seqnum:               0,
		callbackURLRegexp:    callbackURLRegexp,
		followURLRegexp:      followURLRegexp,
		anchorTextRegexp:     anchorTextRegexp,
		subscriber:           subr,
		mu:                   sync.Mutex{},
		duplicates:           map[string]bool{},
		cancelChan:           canc,
		doneListeners:        []chan jobDoneSignal{},
		registerDoneListener: regDoneC,
		doneChan:             make(chan jobDoneSignal),
		randChan:             randChan,
		log:                  nil,
	}
	go randomGenerator(int(nj.opts.MinDelay), int(nj.opts.MaxDelay), randChan)
	nj.retChan <- newJobStatus{
		sub:                  sub,
		subscriber:           subr,
		cancelChan:           canc,
		registerDoneListener: regDoneC,
		err:                  nil,
	}
}

func (s *ideaCrawlerServer) JobManager(newJobChan <-chan newJob, newSubChan <-chan newSub) {
	for {
		select {
		case nj := <-newJobChan:
			s.addNewJob(nj)
		case ns := <-newSubChan:
			job := s.jobs[ns.sub.Subcode]
			if job == nil {
				ns.retChan <- newJobStatus{
					sub:                  pb.Subscription{},
					subscriber:           &subscriber{},
					cancelChan:           nil,
					registerDoneListener: nil,
					err:                  errors.New("Unable to find subcode - " + ns.sub.Subcode),
				}
				continue
			}
			ns.retChan <- newJobStatus{
				sub:                  job.sub,
				subscriber:           job.subscriber,
				cancelChan:           job.cancelChan,
				registerDoneListener: job.registerDoneListener,
				err:                  nil,
			}
		default:
			time.Sleep(50 * time.Millisecond)
		}
		for domainname, job := range s.jobs {
			if job.running {
				continue
			}
			if job.done {
				delete(s.jobs, domainname)
				continue
			}
			if job.prevRun.IsZero() && job.nextRun.IsZero() {
				if job.opts.Firstrun != nil && job.opts.Firstrun.Seconds > 0 {
					job.nextRun = time.Unix(job.opts.Firstrun.Seconds, 0)
				} else {
					job.prevRun = time.Now()
					job.running = true
					go s.RunJob(domainname, job)
					continue
				}
			}
			if job.prevRun.IsZero() {
				if time.Now().After(job.nextRun) {
					job.prevRun = time.Now()
					job.running = true
					go s.RunJob(domainname, job)
					continue
				}
				continue
			}
			if time.Now().After(job.prevRun.Add(job.frequency)) {
				job.prevRun = time.Now()
				job.running = true
				go s.RunJob(domainname, job)
				continue
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (s *ideaCrawlerServer) AddPages(stream pb.IdeaCrawler_AddPagesServer) error {
	pgreq1, err := stream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	if pgreq1 == nil {
		emsg := "Received nil pagereq in AddPages.  Exiting AddPages"
		log.Println(emsg)
		return errors.New(emsg)
	}
	if pgreq1.Sub == nil {
		emsg := fmt.Sprintf("Received pagereq with nil sub object. Exiting AddPages.  PageReq - %v", pgreq1)
		log.Println(emsg)
		return errors.New(emsg)
	}
	retChan := make(chan newJobStatus)
	s.newSubChan <- newSub{*pgreq1.Sub, retChan}
	njs := <-retChan
	if njs.err != nil {
		return njs.err
	}
	jobDoneChan := make(chan jobDoneSignal)
	njs.registerDoneListener <- jobDoneChan
	reqChan := njs.subscriber.reqChan
	reqChan <- *pgreq1
	log.Printf("Adding new page for job '%v': %v", pgreq1.Sub.Subcode, pgreq1.Url)
	for {
		pgreq, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if pgreq == nil {
			emsg := "Received nil pagereq in AddPages.  Exiting AddPages"
			log.Println(emsg)
			return errors.New(emsg)
		}
		select {
		case <-jobDoneChan:
			return nil
		default:
			time.Sleep(10 * time.Millisecond)
		}
		reqChan <- *pgreq
		log.Printf("Adding new page for job '%v': %v", pgreq.Sub.Subcode, pgreq.Url)
	}
}

func (s *ideaCrawlerServer) CancelJob(ctx context.Context, sub *pb.Subscription) (*pb.Status, error) {
	if sub == nil {
		emsg := "Received nil subscription in CancelJob.  Not canceling anything."
		log.Println(emsg)
		return &pb.Status{false, emsg}, errors.New(emsg)
	}
	log.Println("Cancel request received for job:", sub.Subcode)
	retChan := make(chan newJobStatus)
	s.newSubChan <- newSub{*sub, retChan}
	njs := <-retChan
	if njs.err != nil {
		log.Println("ERR - Cancel failed -", njs.err.Error())
		return &pb.Status{false, njs.err.Error()}, njs.err
	}
	njs.cancelChan <- cancelSignal{}
	return &pb.Status{true, ""}, nil
}

func (s *ideaCrawlerServer) GetAnalyzedURLs(sub *pb.Subscription, ostream pb.IdeaCrawler_GetAnalyzedURLsServer) error {
	if sub == nil {
		emsg := "Received nil subscription in GetAnalyzedURLs.  Not requesting analyzed urls."
		log.Println(emsg)
		return errors.New(emsg)
	}
	log.Println("Analyzed urls request received for job:", sub.Subcode)
	retChan := make(chan newJobStatus)
	s.newSubChan <- newSub{*sub, retChan}
	njs := <-retChan
	if njs.err != nil {
		log.Println("ERR - Get analyzed urls request failed -", njs.err.Error())
		return njs.err
	}
	analyzedURLChan := make(chan pb.UrlList, 100)
	stopAnalyzedURLChan := make(chan bool, 3)

	njs.subscriber.analyzedURLConnected = true
	njs.subscriber.analyzedURLChan = analyzedURLChan
	njs.subscriber.stopAnalyzedURLChan = stopAnalyzedURLChan
	log.Println("Analyzed urls request registered")
	for urlList := range njs.subscriber.analyzedURLChan {
		err := ostream.Send(&urlList)
		if err != nil {
			log.Printf("Failed to send analyzed urls to client. No longer trying - %v. Error - %v\n", njs.sub.Subcode, err)
			njs.subscriber.stopAnalyzedURLChan <- true
			return err
		}
	}
	return nil
}

func (s *ideaCrawlerServer) AddDomainAndListen(opts *pb.DomainOpt, ostream pb.IdeaCrawler_AddDomainAndListenServer) error {
	retChan := make(chan newJobStatus)
	s.newJobChan <- newJob{opts, retChan, true}
	njs := <-retChan
	if njs.err != nil {
		return njs.err
	}
	if njs.subscriber.connected == false {
		return errors.New("Subscriber object not created")
	}
	log.Println("Sending subscription object to client:", njs.sub.Subcode)
	// send an empty pagehtml with just the subscription object,  as soon as job starts.
	err := ostream.Send(&pb.PageHTML{
		Success:        true,
		Error:          "subscription.object",
		Sub:            &njs.sub,
		Url:            "",
		Httpstatuscode: sc.Subscription,
		Content:        []byte{},
	})
	if err != nil {
		log.Printf("Failed to send sub object to client. Cancelling job - %v. Error - %v\n", njs.sub.Subcode, err)
		njs.subscriber.stopChan <- true
		return err
	}

	for pagehtml := range njs.subscriber.sendChan {
		err := ostream.Send(&pagehtml)
		if err != nil {
			log.Printf("Failed to send page back to client. No longer trying - %v. Error - %v\n", njs.sub.Subcode, err)
			njs.subscriber.stopChan <- true
			return err
		}
	}
	return nil
}

func (j *job) enqueueLinks(ctx *fetchbot.Context, doc *goquery.Document, urlDepth int32, requestURL *url.URL) {
	j.mu.Lock()
	defer j.mu.Unlock()
	var SendMethod = "GET"
	if j.opts.CheckContent == true {
		SendMethod = "HEAD"
	}
	var urlMap = make(map[string]bool)
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		val, _ := s.Attr("href")
		anchorText := strings.TrimSpace(s.Text())

		// Resolve address
		u, err := requestURL.Parse(val)
		if err != nil {
			j.log.Printf("enqueuelinks: resolve URL %s - %s\n", val, err)
			return
		}
		normFlags := purell.FlagsSafe
		if j.opts.UnsafeNormalizeURL == true {
			normFlags = purell.FlagsSafe | purell.FlagRemoveFragment | purell.FlagRemoveDirectoryIndex
			// Remove query params
			u.RawQuery = ""
		}
		nurl := purell.NormalizeURL(u, normFlags)
		if j.subscriber.analyzedURLConnected == true {
			urlMap[nurl] = true
		}
		var reqMatch = true
		var followMatch = true
		var anchorMatch = true
		if j.callbackURLRegexp != nil && j.callbackURLRegexp.MatchString(nurl) == false {
			reqMatch = false
		}
		if j.followURLRegexp != nil && j.followURLRegexp.MatchString(nurl) == false {
			followMatch = false
		}
		if j.anchorTextRegexp != nil && j.anchorTextRegexp.MatchString(anchorText) == false {
			anchorMatch = false
		}
		if !reqMatch && !followMatch && !anchorMatch {
			return
		}
		if !j.duplicates[nurl] {
			if j.opts.SeedUrl != "" && (!j.opts.FollowOtherDomains && u.Hostname() != j.domainname) {
				j.duplicates[nurl] = true
				return
			}
			parsedURL, err := url.Parse(nurl)
			if err != nil {
				j.log.Println(err)
				return

			}

			cmd := CrawlCommand{
				method:     SendMethod,
				url:        parsedURL,
				metaStr:    ctx.Cmd.(CrawlCommand).MetaStr(),
				urlDepth:   urlDepth,
				anchorText: anchorText,
			}
			//cmd, err := CreateCommand(SendMethod, nurl, "", urlDepth)
			//if err != nil {
			//	job.log.Println(err)
			//	return
			//}
			j.log.Printf("Enqueueing URL: %s", nurl)
			if err := ctx.Q.Send(cmd); err != nil {
				j.log.Printf("error: enqueue head %s - %s", nurl, err)
			} else {
				j.duplicates[nurl] = true
			}
		}
	})
	j.log.Println("Status of analyzed url: ", j.subscriber.analyzedURLConnected)
	if j.subscriber.analyzedURLConnected == true {
		if j.subscriber.connected == false {
			return
		}

		urlList := pb.UrlList{}
		for url := range urlMap {
			urlList.Url = append(urlList.Url, url)
		}
		urlList.MetaStr = ctx.Cmd.(CrawlCommand).MetaStr()
		urlList.UrlDepth = urlDepth

		select {
		case <-j.subscriber.stopAnalyzedURLChan:
			j.subscriber.analyzedURLConnected = false
			return
		case j.subscriber.analyzedURLChan <- urlList:
			return
		}
	}
}

func newServer(newJobChan chan<- newJob, newSubChan chan<- newSub) *ideaCrawlerServer {
	s := new(ideaCrawlerServer)
	s.jobs = make(map[string]*job)
	s.newJobChan = newJobChan
	s.newSubChan = newSubChan
	return s
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	rand.Seed(time.Now().UTC().UnixNano())

	sflag.Parse(&cliParams)
	if cliParams.LogPath != "" {
		err := os.MkdirAll(cliParams.LogPath, 0755)
		if err != nil {
			panic(err)
		}

		logFP, err := os.Create(path.Join(cliParams.LogPath, "master.log"))
		if err != nil {
			panic(err)
		}
		log.SetOutput(logFP)
	}

	log.Println(cliParams)
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", cliParams.ListenAddress, cliParams.ListenPort))
	if err != nil {
		log.Printf("failed to listen: %v", err)
		os.Exit(1)
	}

	defer fmt.Println("Exiting crawler. Bye")
	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)
	newJobChan := make(chan newJob)
	newSubChan := make(chan newSub)
	newsrv := newServer(newJobChan, newSubChan)
	pb.RegisterIdeaCrawlerServer(grpcServer, newsrv)
	go newsrv.JobManager(newJobChan, newSubChan)
	grpcServer.Serve(lis)
}
