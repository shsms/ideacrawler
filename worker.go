package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"github.com/ideas2it/ideacrawler/chromeclient"
	pb "github.com/ideas2it/ideacrawler/protofiles"
	sc "github.com/ideas2it/ideacrawler/statuscodes"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type clusterWorkerListener struct {
	conn      net.Conn
	connected bool
	closeChan chan bool
	closed    bool
}

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

type ideaCrawlerWorker struct {
	mode       int
	jobs       map[string]*job
	newJobChan chan<- newJob
	newSubChan chan<- newSub
	ccl        *chromeclient.ChromeClient
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
func (s *ideaCrawlerWorker) addNewJob(nj newJob) {
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
	var callbackURLRegexp, followURLRegexp, callbackAnchorTextRegexp *regexp.Regexp
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
	if len(nj.opts.CallbackAnchorTextRegexp) > 0 {
		callbackAnchorTextRegexp, err = regexp.Compile(nj.opts.CallbackAnchorTextRegexp)
		if err != nil {
			nj.retChan <- jobStatusFailureMessage(fmt.Errorf("AnchorTextRegexp doesn't compile - %s - %s", nj.opts.CallbackAnchorTextRegexp, err))
			return
		}
	}
	canc := make(chan cancelSignal)
	regDoneC := make(chan chan jobDoneSignal)
	randChan := make(chan int, 5)
	s.jobs[sub.Subcode] = &job{
		domainname:               domainname,
		opts:                     nj.opts,
		sub:                      sub,
		prevRun:                  time.Time{},
		nextRun:                  time.Time{},
		frequency:                freq,
		runNumber:                0,
		running:                  false,
		done:                     false,
		seqnum:                   0,
		callbackURLRegexp:        callbackURLRegexp,
		followURLRegexp:          followURLRegexp,
		callbackAnchorTextRegexp: callbackAnchorTextRegexp,
		subscriber:               subr,
		mu:                       sync.Mutex{},
		duplicates:               map[string]bool{},
		cancelChan:               canc,
		doneListeners:            []chan jobDoneSignal{},
		registerDoneListener:     regDoneC,
		doneChan:                 make(chan jobDoneSignal),
		randChan:                 randChan,
		log:                      nil,
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

func (s *ideaCrawlerWorker) jobManager(newJobChan <-chan newJob, newSubChan <-chan newSub) {
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

func (s *ideaCrawlerWorker) AddPages(stream pb.IdeaCrawler_AddPagesServer) error {
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

func (s *ideaCrawlerWorker) CancelJob(ctx context.Context, sub *pb.Subscription) (*pb.Status, error) {
	if sub == nil {
		emsg := "Received nil subscription in CancelJob.  Not canceling anything."
		log.Println(emsg)
		return &pb.Status{Success: false, Error: emsg}, errors.New(emsg)
	}
	log.Println("Cancel request received for job:", sub.Subcode)
	retChan := make(chan newJobStatus)
	s.newSubChan <- newSub{*sub, retChan}
	njs := <-retChan
	if njs.err != nil {
		log.Println("ERR - Cancel failed -", njs.err.Error())
		return &pb.Status{Success: false, Error: njs.err.Error()}, njs.err
	}
	njs.cancelChan <- cancelSignal{}
	return &pb.Status{Success: true, Error: ""}, nil
}

func (s *ideaCrawlerWorker) GetAnalyzedURLs(sub *pb.Subscription, ostream pb.IdeaCrawler_GetAnalyzedURLsServer) error {
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

func (s *ideaCrawlerWorker) AddDomainAndListen(opts *pb.DomainOpt, ostream pb.IdeaCrawler_AddDomainAndListenServer) error {
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

func (l *clusterWorkerListener) Accept() (net.Conn, error) {
	if l.closed == true {
		return nil, errors.New("Listener closed")
	}
	if l.connected == false {
		l.connected = true
		return l.conn, nil
	}
	for l.closed == false {
		time.Sleep(100 * time.Millisecond)
	}
	return nil, errors.New("Listener closed")
}

func (l *clusterWorkerListener) Close() error {
	if l.closed == true {
		return nil
	}
	close(l.closeChan)
	return nil
}

func (l *clusterWorkerListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}

func newClusterWorkerListener() net.Listener {
	var err error
	l := &clusterWorkerListener{}
	l.conn, err = net.Dial("tcp", cliParams.Servers)
	if err != nil {
		log.Fatal("Unable to connect to servers")
	}
	l.closeChan = make(chan bool)
	return l
}

func newStandaloneListener() net.Listener {
	lis, err := net.Listen("tcp", cliParams.ClientListenAddress)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		os.Exit(1)
	}
	log.Println("Listening on", cliParams.ClientListenAddress)
	return lis
}

func newServer(newJobChan chan<- newJob, newSubChan chan<- newSub) *ideaCrawlerWorker {
	s := new(ideaCrawlerWorker)
	s.jobs = make(map[string]*job)
	s.newJobChan = newJobChan
	s.newSubChan = newSubChan
	return s
}

func startCrawlerWorker(mode int) {
	var lis net.Listener
	if mode == modeStandalone {
		lis = newStandaloneListener()
	} else if mode == modeWorker {
		lis = newClusterWorkerListener()
	}
	defer log.Println("Exiting crawler. Bye")
	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)
	newJobChan := make(chan newJob)
	newSubChan := make(chan newSub)
	newsrv := newServer(newJobChan, newSubChan)
	pb.RegisterIdeaCrawlerServer(grpcServer, newsrv)
	go newsrv.jobManager(newJobChan, newSubChan)
	grpcServer.Serve(lis)
}