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
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/cookiejar"
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
	"github.com/antchfx/xpath"
	htmlquery "github.com/antchfx/xquery/html"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/ideas2it/ideacrawler/chromeclient"
	pb "github.com/ideas2it/ideacrawler/protofiles"
	sc "github.com/ideas2it/ideacrawler/statuscodes"
	"github.com/shsms-i2i/sflag"
	"golang.org/x/net/context"
	"golang.org/x/sync/semaphore"
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

type job struct {
	domainname        string
	opts              *pb.DomainOpt
	sub               pb.Subscription
	prevRun           time.Time
	nextRun           time.Time
	frequency         time.Duration
	runNumber         int32
	running           bool
	done              bool
	seqnum            int32
	callbackURLRegexp *regexp.Regexp
	followURLRegexp   *regexp.Regexp
	anchorTextRegexp  *regexp.Regexp
	subscriber        *subscriber
	mu                sync.Mutex
	duplicates        map[string]bool
	cancelChan        chan cancelSignal

	// there will be a goroutine started inside RunJob that will listen on registerDoneListener for
	// DoneListeners.  There will be a despatcher goroutine that will forward the doneChan info to
	// all doneListeners.
	// TODO: Needs to be replaced with close(chan) signals.
	doneListeners        []chan jobDoneSignal
	registerDoneListener chan chan jobDoneSignal
	doneChan             chan jobDoneSignal
	randChan             <-chan int // for random number between min and max norm distributed

	log *log.Logger
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

func (s *ideaCrawlerServer) addNewJob(nj newJob) {
	log.Println("Received new job", nj.opts.SeedUrl)
	domainname, err := domainNameFromURL(nj.opts.SeedUrl)
	if err != nil {
		nj.retChan <- newJobStatus{
			sub:                  pb.Subscription{},
			subscriber:           &subscriber{},
			cancelChan:           nil,
			registerDoneListener: nil,
			err:                  err,
		}
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
		nj.retChan <- newJobStatus{
			sub:                  pb.Subscription{},
			subscriber:           &subscriber{},
			cancelChan:           nil,
			registerDoneListener: nil,
			err:                  errors.New("Bad value for DomainOpt.Frequency field - " + domainname + " - " + err.Error()),
		}
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
			nj.retChan <- newJobStatus{
				sub:                  pb.Subscription{},
				subscriber:           &subscriber{},
				cancelChan:           nil,
				registerDoneListener: nil,
				err:                  errors.New("CallbackUrlRegexp doesn't compile - '" + nj.opts.CallbackUrlRegexp + "' - " + err.Error()),
			}
			return
		}
	}
	if len(nj.opts.FollowUrlRegexp) > 0 {
		followURLRegexp, err = regexp.Compile(nj.opts.FollowUrlRegexp)
		if err != nil {
			nj.retChan <- newJobStatus{
				sub:                  pb.Subscription{},
				subscriber:           &subscriber{},
				cancelChan:           nil,
				registerDoneListener: nil,
				err:                  errors.New("FollowUrlRegexp doesn't compile - '" + nj.opts.FollowUrlRegexp + "' - " + err.Error()),
			}
			return
		}
	}
	if len(nj.opts.AnchorTextRegexp) > 0 {
		anchorTextRegexp, err = regexp.Compile(nj.opts.AnchorTextRegexp)
		if err != nil {
			nj.retChan <- newJobStatus{
				sub:                  pb.Subscription{},
				subscriber:           &subscriber{},
				cancelChan:           nil,
				registerDoneListener: nil,
				err:                  errors.New("AnchorTextRegexp doesn't compile - '" + nj.opts.AnchorTextRegexp + "' - " + err.Error()),
			}
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

func (s *ideaCrawlerServer) RunJob(subID string, j *job) {
	var logfile = "/dev/stdout"
	if cliParams.LogPath != "" {
		logfile = path.Join(cliParams.LogPath, "job-"+subID+".log")
	}
	logFP, err := os.Create(logfile)
	if err != nil {
		log.Printf("Unable to create logfile %v. not proceeding with job.", logfile)
		return
	}
	j.log = log.New(logFP, subID+": ", log.LstdFlags|log.Lshortfile)
	j.log.Println("starting job -", subID)
	go func() {
		j.doneListeners = []chan jobDoneSignal{}
	registerDoneLoop:
		for {
			select {
			case vv := <-j.registerDoneListener:
				j.doneListeners = append(j.doneListeners, vv)
			case jds := <-j.doneChan:
				for _, ww := range j.doneListeners {
					ww <- jds
				}
				break registerDoneLoop
			}
		}
	}()
	defer func() {
		close(j.subscriber.sendChan)
		j.running = false

		j.done = true
		j.doneChan <- jobDoneSignal{}
		j.log.Println("stopping job -", subID)
	}()
	j.runNumber++

	sendPageHTML := func(ctx *fetchbot.Context, phtml pb.PageHTML) {
		if j.subscriber.connected == false {
			return
		}
		select {
		case <-j.subscriber.stopChan:
			j.subscriber.connected = false
			if ctx != nil && j.opts.CancelOnDisconnect {
				j.log.Printf("Lost client, cancelling queue.\n")
				j.cancelChan <- cancelSignal{}
			} else {
				j.log.Printf("Lost client\n")
			}
			return
		case j.subscriber.sendChan <- phtml:
			return
		}
	}

	mux := fetchbot.NewMux()
	mux.HandleErrors(fetchbot.HandlerFunc(func(ctx *fetchbot.Context, res *http.Response, err error) {
		j.log.Printf("ERR - fetch error : %s\n", err.Error())
		phtml := pb.PageHTML{
			Success: false,
			Error:   err.Error(),
			Sub:     &pb.Subscription{},
			//TODO -- Check if ctx always has the correct URL
			Url:            ctx.Cmd.URL().String(),
			Httpstatuscode: sc.FetchbotError,
			Content:        []byte{},
			MetaStr:        ctx.Cmd.(CrawlCommand).MetaStr(),
			UrlDepth:       ctx.Cmd.(CrawlCommand).URLDepth(),
		}
		sendPageHTML(ctx, phtml)
		return
	}))
	// Handle GET requests for html responses, to parse the body and enqueue all links as HEAD
	// requests.
	mux.Response().Method("GET").ContentType("text/html").Handler(fetchbot.HandlerFunc(
		func(ctx *fetchbot.Context, res *http.Response, err error) {
			requestURL := res.Request.URL
			ccmd := ctx.Cmd.(CrawlCommand)
			anchorText := ccmd.anchorText
			if ccmd.noCallback == true {
				return
			}
			if res.StatusCode > 399 && res.StatusCode < 600 {
				j.log.Printf("STATUS - %s %s - %d : %s\n", ctx.Cmd.Method(), ctx.Cmd.URL(), res.StatusCode, res.Status)
				if ccmd.URLDepth() == 0 {
					phtml := pb.PageHTML{
						Success: false,
						Error:   res.Status,
						Sub:     &pb.Subscription{},
						//Url:            ctx.Cmd.URL().String(),
						Url:            requestURL.String(),
						Httpstatuscode: int32(res.StatusCode),
						Content:        []byte{},
						MetaStr:        ccmd.MetaStr(),
						UrlDepth:       ccmd.URLDepth(),
					}
					sendPageHTML(ctx, phtml)
				}
				return
			}
			pageBody, err := ioutil.ReadAll(res.Body)
			if err != nil {
				emsg := fmt.Sprintf("ERR - %s %s - %s", ctx.Cmd.Method(), ctx.Cmd.URL(), err)
				j.log.Println(emsg)
				phtml := pb.PageHTML{
					Success:        false,
					Error:          emsg,
					Sub:            &pb.Subscription{},
					Url:            requestURL.String(),
					Httpstatuscode: sc.ResponseError,
					MetaStr:        ctx.Cmd.(CrawlCommand).MetaStr(),
					Content:        []byte{},
					UrlDepth:       ccmd.URLDepth(),
				}
				sendPageHTML(ctx, phtml)
				return
			}

			// check if still logged in
			if j.opts.Login && j.opts.CheckLoginAfterEachPage && j.opts.LoginSuccessCheck != nil {
				doc, _ := htmlquery.Parse(bytes.NewBuffer(pageBody))
				expr, _ := xpath.Compile(j.opts.LoginSuccessCheck.Key) // won't lead to error now, because this is the second time this is happening.

				iter := expr.Evaluate(htmlquery.CreateXPathNavigator(doc)).(*xpath.NodeIterator)
				iter.MoveNext()
				if strings.ToLower(iter.Current().Value()) != strings.ToLower(j.opts.LoginSuccessCheck.Value) {
					errMsg := fmt.Sprintf("Not logged in anymore: In xpath '%s', expected '%s', but got '%s'. Not continuing.", j.opts.LoginSuccessCheck.Key, j.opts.LoginSuccessCheck.Value, iter.Current().Value())
					j.log.Println(errMsg)
					phtml := pb.PageHTML{
						Success:        false,
						Error:          errMsg,
						Sub:            nil, //no subscription object
						Url:            "",
						Httpstatuscode: sc.NolongerLoggedIn,
						Content:        []byte{},
						UrlDepth:       ccmd.URLDepth(),
					}
					sendPageHTML(ctx, phtml)
					j.cancelChan <- cancelSignal{}
					if cliParams.SaveLoginPages != "" {
						err := ioutil.WriteFile(path.Join(cliParams.SaveLoginPages, "loggedout.html"), pageBody, 0755)
						if err != nil {
							j.log.Println("ERR - unable to save loggedout file:", err)
						}
					}
					j.done = true
					return
				}
			}

			// Enqueue all links as HEAD requests, if they match followUrlRegexp
			if j.opts.NoFollow == false && (j.followURLRegexp == nil || j.followURLRegexp.MatchString(ctx.Cmd.URL().String()) == true) && (j.opts.Depth < 0 || ccmd.URLDepth() < j.opts.Depth) {
				// Process the body to find the links
				doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(pageBody)))
				if err != nil {
					emsg := fmt.Sprintf("ERR - %s %s - %s\n", ctx.Cmd.Method(), ctx.Cmd.URL(), err)
					j.log.Println(emsg)
					phtml := pb.PageHTML{
						Success:        false,
						Error:          emsg,
						Sub:            &pb.Subscription{},
						Url:            "",
						Httpstatuscode: sc.FollowParseError,
						Content:        []byte{},
						UrlDepth:       ccmd.URLDepth(),
					}
					sendPageHTML(ctx, phtml)
					return
				}
				j.enqueueLinks(ctx, doc, ccmd.URLDepth()+1, requestURL)
				j.log.Println("Enqueued", ctx.Cmd.URL().String())
			}

			var callbackPage = false

			// Callback if depth = 0
			if j.opts.CallbackSeedUrl == true && ccmd.URLDepth() == 0 {
				callbackPage = true
			}

			if len(j.opts.CallbackUrlRegexp) == 0 && len(j.opts.CallbackXpathMatch) == 0 && len(j.opts.CallbackXpathRegexp) == 0 {
				callbackPage = true
			}

			if j.callbackURLRegexp != nil && j.callbackURLRegexp.MatchString(ctx.Cmd.URL().String()) == false {
				j.log.Printf("Url '%v' did not match callbackRegexp '%v'\n", ctx.Cmd.URL().String(), j.callbackURLRegexp)
			} else if j.callbackURLRegexp != nil {
				callbackPage = true
			}

			if j.anchorTextRegexp != nil && j.anchorTextRegexp.MatchString(anchorText) == false {
				j.log.Printf("Anchor Text '%v' did not match anchorTextRegexp '%v'\n", anchorText, j.anchorTextRegexp)
			} else if j.anchorTextRegexp != nil {
				callbackPage = true
			}

			if callbackPage == false && len(j.opts.CallbackXpathMatch) > 0 {
				passthru := true
				doc, _ := htmlquery.Parse(bytes.NewBuffer(pageBody))
				for _, xm := range j.opts.CallbackXpathMatch {
					expr, _ := xpath.Compile(xm.Key)

					iter := expr.Evaluate(htmlquery.CreateXPathNavigator(doc)).(*xpath.NodeIterator)
					iter.MoveNext()
					if iter.Current().Value() != xm.Value {
						passthru = false
						j.log.Printf("Url '%v' did not have '%v' at xpath '%v'\n", ctx.Cmd.URL().String(), xm.Value, xm.Key)
						break
					}
				}
				if passthru {
					callbackPage = true
				}
			}

			if callbackPage == false && len(j.opts.CallbackXpathRegexp) > 0 {
				passthru := true
				doc, _ := htmlquery.Parse(bytes.NewBuffer(pageBody))
				for _, xm := range j.opts.CallbackXpathRegexp {
					expr, _ := xpath.Compile(xm.Key)

					iter := expr.Evaluate(htmlquery.CreateXPathNavigator(doc)).(*xpath.NodeIterator)
					iter.MoveNext()
					if iter.Current().Value() != xm.Value {
						passthru = false
						j.log.Printf("Url '%v' did not have '%v' at xpath '%v'\n", ctx.Cmd.URL().String(), xm.Value, xm.Key)
						break
					}
				}
				if passthru {
					callbackPage = true
				}
			}

			if callbackPage == false {
				return // no need to send page back to client.
			}
			shippingURL := ""
			if strings.HasPrefix(ctx.Cmd.URL().Scheme, "crawljs") == true {
				shippingURL, err = url.QueryUnescape(ctx.Cmd.URL().RawQuery)
				if err != nil {
					j.log.Printf("Unable to UnEscape RawQuery - %v, err: %v\n", ctx.Cmd.URL().RawQuery, err)
					shippingURL = ctx.Cmd.URL().String()
				}
			} else {
				shippingURL = ctx.Cmd.URL().String()
			}
			j.log.Println("Shipping", shippingURL)

			sub := j.sub

			phtml := pb.PageHTML{
				Success:        true,
				Error:          "",
				Sub:            &sub,
				Url:            shippingURL,
				Httpstatuscode: int32(res.StatusCode),
				Content:        pageBody,
				MetaStr:        ccmd.MetaStr(),
				UrlDepth:       ccmd.URLDepth(),
				AnchorText:     anchorText,
			}
			sendPageHTML(ctx, phtml)
		}))

	// Handle HEAD requests for html responses coming from the source host - we don't want
	// to crawl links from other hosts.
	mux.Response().Method("HEAD").ContentType("text/html").Handler(fetchbot.HandlerFunc(
		func(ctx *fetchbot.Context, res *http.Response, err error) {
			cmd := CrawlCommand{
				method:     "GET",
				url:        ctx.Cmd.URL(),
				noCallback: false,
				metaStr:    ctx.Cmd.(CrawlCommand).MetaStr(),
			}
			if err := ctx.Q.Send(cmd); err != nil {
				j.log.Printf("ERR - %s %s - %s\n", ctx.Cmd.Method(), ctx.Cmd.URL(), err)
			}
		}))

	f := fetchbot.New(mux)
	j.log.Printf("MaxConcurrentRequests=%v\n", int64(j.opts.MaxConcurrentRequests))
	var networkTransport http.RoundTripper
	if j.opts.NetworkIface != "" && cliParams.DialAddress != "" {
		localAddress := &net.TCPAddr{
			IP:   net.ParseIP(cliParams.DialAddress), // a secondary local IP I assigned to me
			Port: 80,
		}
		networkTransport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
				LocalAddr: localAddress,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}
	if j.opts.Chrome == false && j.opts.Login == false {
		gCurCookieJar, _ := cookiejar.New(nil)
		httpClient := &http.Client{
			Transport:     networkTransport,
			CheckRedirect: nil,
			Jar:           gCurCookieJar,
		}
		if j.opts.Prefetch == true {
			httpClient.Transport = &httpcache.Transport{
				Transport:           networkTransport,
				Cache:               diskcache.New("/tmp/ideacache/" + j.sub.Subcode),
				MarkCachedResponses: true,
			}
		}
		f.HttpClient = &Doer{httpClient, j, semaphore.NewWeighted(int64(j.opts.MaxConcurrentRequests)), s}
	}
	if j.opts.Chrome == false && j.opts.Login == true {
		// create our own httpClient and attach a cookie jar to it,
		// login using that client to the site if required,
		// check if login succeeded,
		// then give that client to fetchbot's fetcher.
		gCurCookieJar, _ := cookiejar.New(nil)
		httpClient := &http.Client{
			Transport:     networkTransport,
			CheckRedirect: nil,
			Jar:           gCurCookieJar,
		}
		if j.opts.Prefetch == true {
			httpClient.Transport = &httpcache.Transport{
				Transport:           networkTransport,
				Cache:               diskcache.New("/tmp/ideacache/" + j.sub.Subcode),
				MarkCachedResponses: true,
			}
		}
		var payload = make(url.Values)
		if j.opts.LoginParseFields == true {
			afterTime := j.opts.MinDelay
			if afterTime < 5 {
				afterTime = 5
			}
			if j.opts.MaxDelay > j.opts.MinDelay {
				afterTime = int32(<-j.randChan)
				j.log.Printf("Next delay - %v\n", time.Duration(afterTime)*time.Second)
			}
			after := time.After(time.Duration(afterTime) * time.Second)

			httpReq, err := http.NewRequest("GET", j.opts.LoginUrl, nil)
			if err != nil {
				j.log.Println(err)
				return
			}
			httpReq.Header.Set("User-Agent", j.opts.Useragent)

			httpResp, err := httpClient.Do(httpReq)
			if err != nil {
				j.log.Println(err)
				return
			}
			pageBody, err := ioutil.ReadAll(httpResp.Body)
			if err != nil {
				j.log.Println("Unable to read http response:", err)
				return
			}
			if cliParams.SaveLoginPages != "" {
				err := ioutil.WriteFile(path.Join(cliParams.SaveLoginPages, "login.html"), pageBody, 0755)
				if err != nil {
					j.log.Println("ERR - Unable to save login file:", err)
				}
			}

			doc, _ := htmlquery.Parse(bytes.NewBuffer(pageBody))
			for _, kvp := range j.opts.LoginParseXpath {
				expr, err := xpath.Compile(kvp.Value)
				if err != nil {
					j.log.Println(err)
					return
				}

				iter := expr.Evaluate(htmlquery.CreateXPathNavigator(doc)).(*xpath.NodeIterator)
				iter.MoveNext()
				payload.Set(kvp.Key, iter.Current().Value())
			}
			<-after
		}

		for _, kvp := range j.opts.LoginPayload {
			payload.Set(kvp.Key, kvp.Value)
		}
		httpResp, err := httpClient.PostForm(j.opts.LoginUrl, payload)
		if err != nil {
			j.log.Println(err)
			return
		}
		pageBody, err := ioutil.ReadAll(httpResp.Body)
		if cliParams.SaveLoginPages != "" {
			err := ioutil.WriteFile(path.Join(cliParams.SaveLoginPages, "loggedin.html"), pageBody, 0755)
			if err != nil {
				j.log.Println("ERR - Unable to save loggedin file:", err)
			}

		}
		doc, _ := htmlquery.Parse(bytes.NewBuffer(pageBody))
		if j.opts.LoginSuccessCheck != nil {
			expr, err := xpath.Compile(j.opts.LoginSuccessCheck.Key)
			if err != nil {
				j.log.Println(err)
				return
			}
			iter := expr.Evaluate(htmlquery.CreateXPathNavigator(doc)).(*xpath.NodeIterator)
			iter.MoveNext()
			if strings.ToLower(iter.Current().Value()) != strings.ToLower(j.opts.LoginSuccessCheck.Value) {
				errMsg := fmt.Sprintf("Login failed: In xpath '%s', expected '%s', but got '%s'. Not proceeding.", j.opts.LoginSuccessCheck.Key, j.opts.LoginSuccessCheck.Value, iter.Current().Value())
				j.log.Println(errMsg)
				phtml := pb.PageHTML{
					Success:        false,
					Error:          errMsg,
					Sub:            nil, //no subscription object
					Url:            "",
					Httpstatuscode: sc.LoginFailed,
					Content:        []byte{},
				}
				sendPageHTML(nil, phtml)
				return
			}
			phtml := pb.PageHTML{
				Success:        true,
				Error:          "",
				Sub:            nil, //no subscription object
				Url:            "",
				Httpstatuscode: sc.LoginSuccess,
				Content:        []byte{},
			}
			sendPageHTML(nil, phtml)
			j.log.Printf("Logged in. Found '%v' in '%v'\n", j.opts.LoginSuccessCheck.Value, j.opts.LoginSuccessCheck.Key)
		}
		f.HttpClient = &Doer{httpClient, j, semaphore.NewWeighted(int64(j.opts.MaxConcurrentRequests)), s}
	}

	if j.opts.Chrome == true && j.opts.Login == true {
		j.opts.Impolite = true // Always impolite in Chrome mode.
		if j.opts.ChromeBinary == "" {
			j.opts.ChromeBinary = "/usr/lib64/chromium-browser/headless_shell"
		}
		cl := chromeclient.NewChromeClient(j.opts.ChromeBinary)
		if j.opts.DomLoadTime > 0 {
			cl.SetDomLoadTime(j.opts.DomLoadTime)
		}
		err := cl.Start()
		if err != nil {
			j.log.Println("Unable to start chrome:", err)
			return
		}
		defer cl.Stop()
		urlobj, _ := url.Parse(j.opts.LoginUrl)
		req := &http.Request{
			URL: urlobj,
		}
		loginResp, err := cl.Do(req)
		if err != nil {
			j.log.Println("Http request to chrome failed:", err)
			return
		}
		loginBody, err := ioutil.ReadAll(loginResp.Body)
		if err != nil {
			j.log.Println("Unable to read http response", err)
			return
		}
		if cliParams.SaveLoginPages != "" {
			err := ioutil.WriteFile(path.Join(cliParams.SaveLoginPages, "login.html"), loginBody, 0755)
			if err != nil {
				j.log.Println("ERR - Unable to save login file:", err)
			}
		}
		loginjs := &url.URL{
			Scheme:   "crawljs-jscript",
			Host:     j.opts.LoginUrl,
			Path:     url.PathEscape(j.opts.LoginJS), // js string
			RawQuery: url.QueryEscape(j.opts.LoginUrl),
		}
		loginJsReq := &http.Request{
			URL: loginjs,
		}
		loggedInResp, err := cl.Do(loginJsReq)
		loggedInBody, err := ioutil.ReadAll(loggedInResp.Body)
		if cliParams.SaveLoginPages != "" {
			err := ioutil.WriteFile(path.Join(cliParams.SaveLoginPages, "loggedin.html"), loggedInBody, 0755)
			if err != nil {
				j.log.Println("ERR - Unable to save loggedin file:", err)
			}

		}
		doc, _ := htmlquery.Parse(bytes.NewBuffer(loggedInBody))
		if j.opts.LoginSuccessCheck != nil {
			expr, err := xpath.Compile(j.opts.LoginSuccessCheck.Key)
			if err != nil {
				j.log.Printf("Unable to compile xpath - %v; err:%v\n", j.opts.LoginSuccessCheck.Key, err)
				return
			}
			iter := expr.Evaluate(htmlquery.CreateXPathNavigator(doc)).(*xpath.NodeIterator)
			iter.MoveNext()
			if iter.Current().Value() != j.opts.LoginSuccessCheck.Value {
				errMsg := fmt.Sprintf("Login failed: In xpath '%s', expected '%s', but got '%s'. Not proceeding.", j.opts.LoginSuccessCheck.Key, j.opts.LoginSuccessCheck.Value, iter.Current().Value())
				j.log.Println(errMsg)
				phtml := pb.PageHTML{
					Success:        false,
					Error:          errMsg,
					Sub:            nil, //no subscription object
					Url:            "",
					Httpstatuscode: sc.LoginFailed,
					Content:        []byte{},
				}
				sendPageHTML(nil, phtml)
				return
			}
			phtml := pb.PageHTML{
				Success:        true,
				Error:          "",
				Sub:            nil, //no subscription object
				Url:            "",
				Httpstatuscode: sc.LoginSuccess,
				Content:        []byte{},
			}
			sendPageHTML(nil, phtml)
			j.log.Printf("Logged in. Found '%v' in '%v'\n", j.opts.LoginSuccessCheck.Value, j.opts.LoginSuccessCheck.Key)
		}
		f.HttpClient = &Doer{cl, j, semaphore.NewWeighted(int64(j.opts.MaxConcurrentRequests)), s}
	}

	if j.opts.Chrome == true && j.opts.Login == false {
		j.opts.Impolite = true // Always impolite in Chrome mode.
		if j.opts.ChromeBinary == "" {
			j.opts.ChromeBinary = "/usr/lib64/chromium-browser/headless_shell"
		}
		cl := chromeclient.NewChromeClient(j.opts.ChromeBinary)
		if j.opts.DomLoadTime > 0 {
			cl.SetDomLoadTime(j.opts.DomLoadTime)
		}
		err := cl.Start()
		if err != nil {
			j.log.Println("Unable to start chrome:", err)
			return
		}
		defer cl.Stop()
		f.HttpClient = &Doer{cl, j, semaphore.NewWeighted(int64(j.opts.MaxConcurrentRequests)), s}
	}

	f.DisablePoliteness = j.opts.Impolite
	// minimal crawl delay; actual randomized delay is implemented in IdeaCrawlDoer's Do method.
	f.CrawlDelay = 50 * time.Millisecond
	if j.opts.MaxIdleTime < 600 {
		f.WorkerIdleTTL = 600 * time.Second
	} else {
		f.WorkerIdleTTL = time.Duration(j.opts.MaxIdleTime) * time.Second
	}
	f.AutoClose = true
	f.UserAgent = j.opts.Useragent

	//TODO: hipri: create goroutine to listen for new PageRequest objects

	q := f.Start()

	// handle cancel requests
	go func() {
		jobDoneChan := make(chan jobDoneSignal)
		j.registerDoneListener <- jobDoneChan
		select {
		case <-jobDoneChan:
			return
		case <-j.cancelChan:
			q.Cancel()
			j.log.Println("Cancelled job:", subID)
		}
	}()

	// handle stuff coming through the addPage function
	go func() {
		jobDoneChan := make(chan jobDoneSignal)
		j.registerDoneListener <- jobDoneChan
	handlePagesLoop:
		for {
			select {
			case pr := <-j.subscriber.reqChan: // TODO: No URL normalization if added through this method?
				switch pr.Reqtype {
				case pb.PageReqType_GET:
					// TODO:  add error checking for error from Send functions
					cmd, err := newCrawlCommand("GET", pr.Url, pr.MetaStr, 0)
					if err != nil {
						j.log.Println(err)
						return
					}
					err = q.Send(cmd)
					if err != nil {
						j.log.Println(err)
						return
					}
				case pb.PageReqType_HEAD:
					cmd, err := newCrawlCommand("HEAD", pr.Url, pr.MetaStr, 0)
					if err != nil {
						j.log.Println(err)
						return
					}
					err = q.Send(cmd)
					if err != nil {
						j.log.Println(err)
						return
					}
				case pb.PageReqType_BUILTINJS:
					prURL, err := url.Parse(pr.Url)
					if err != nil {
						j.log.Println(err)
						return
					}
					cmd := CrawlCommand{
						method: "GET",
						url: &url.URL{
							Scheme:   "crawljs-builtinjs",
							Host:     prURL.Host,
							Path:     url.PathEscape(pr.Js), // url command name
							RawQuery: url.QueryEscape(pr.Url),
						},
						noCallback: pr.NoCallback,
						metaStr:    pr.MetaStr,
						urlDepth:   0,
					}
					err = q.Send(cmd)
					if err != nil {
						j.log.Println(err)
						return
					}
				case pb.PageReqType_JSCRIPT:
					prURL, err := url.Parse(pr.Url)
					if err != nil {
						j.log.Println(err)
						return
					}
					cmd := CrawlCommand{
						method: "GET",
						url: &url.URL{
							Scheme:   "crawljs-jscript",
							Host:     prURL.Host,
							Path:     url.PathEscape(pr.Js), // url command name
							RawQuery: url.QueryEscape(pr.Url),
						},
						noCallback: pr.NoCallback,
						metaStr:    pr.MetaStr,
						urlDepth:   0,
					}
					err = q.Send(cmd)
					if err != nil {
						j.log.Println(err)
						return
					}
				}
				j.log.Println("Enqueued page:", pr.Url)
			case <-jobDoneChan:
				break handlePagesLoop
			}
		}
	}()
	if len(j.opts.SeedUrl) > 0 {
		j.mu.Lock()
		j.duplicates[j.opts.SeedUrl] = true
		j.mu.Unlock()
		cmd, err := newCrawlCommand("GET", j.opts.SeedUrl, "", 0)
		if err != nil {
			j.log.Println(err)
			return
		}
		err = q.Send(cmd)
		if err != nil {
			j.log.Println(err)
			return
		}
	}
	q.Block()
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
