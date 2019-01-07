package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/fetchbot"
	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/xpath"
	htmlquery "github.com/antchfx/xquery/html"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/ideas2it/ideacrawler/chromeclient"
	pb "github.com/ideas2it/ideacrawler/protofiles"
	sc "github.com/ideas2it/ideacrawler/statuscodes"
	"golang.org/x/sync/semaphore"
)

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
	cl                   *chromeclient.ChromeClient
	log                  *log.Logger
}

func (j *job) sendPageHTML(ctx *fetchbot.Context, phtml pb.PageHTML) {
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

func (j *job) setupJobLogger(subID string) error {
	var logfile = "/dev/stdout"
	if cliParams.LogPath != "" {
		logfile = path.Join(cliParams.LogPath, "job-"+subID+".log")
	}
	logFP, err := os.Create(logfile)
	if err != nil {
		log.Printf("Unable to create logfile %v. not proceeding with job.", logfile)
		return err
	}
	j.log = log.New(logFP, subID+": ", log.LstdFlags|log.Lshortfile)
	return nil
}

// TODO: replace with goroutines listen for close of doneChan, instead
// of having a custom signal broadcaster.
func (j *job) doneListenersBroadcaster() {
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
}

func (j *job) fetchErrorHandler(ctx *fetchbot.Context, res *http.Response, err error) {
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
	j.sendPageHTML(ctx, phtml)
	return
}

func (j *job) fetchHTTPGetHandler(ctx *fetchbot.Context, res *http.Response, err error) {
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
				Success:        false,
				Error:          res.Status,
				Sub:            &pb.Subscription{},
				Url:            requestURL.String(),
				Httpstatuscode: int32(res.StatusCode),
				Content:        []byte{},
				MetaStr:        ccmd.MetaStr(),
				UrlDepth:       ccmd.URLDepth(),
			}
			j.sendPageHTML(ctx, phtml)
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
		j.sendPageHTML(ctx, phtml)
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
			j.sendPageHTML(ctx, phtml)
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
			j.sendPageHTML(ctx, phtml)
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
	j.sendPageHTML(ctx, phtml)
}

func (j *job) fetchHTTPHeadHandler(ctx *fetchbot.Context, res *http.Response, err error) {
	cmd := CrawlCommand{
		method:     "GET",
		url:        ctx.Cmd.URL(),
		noCallback: false,
		metaStr:    ctx.Cmd.(CrawlCommand).MetaStr(),
	}
	if err := ctx.Q.Send(cmd); err != nil {
		j.log.Printf("ERR - %s %s - %s\n", ctx.Cmd.Method(), ctx.Cmd.URL(), err)
	}
}

func (j *job) makeNetworkTransport() http.RoundTripper {
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
	return networkTransport
}

func (s *ideaCrawlerServer) makeHTTPClientRawDirect(j *job, networkTransport http.RoundTripper) (fetchbot.Doer, error) {
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
	return &Doer{httpClient, j, semaphore.NewWeighted(int64(j.opts.MaxConcurrentRequests)), s}, nil
}

func (s *ideaCrawlerServer) makeHTTPClientLoginDirect(j *job, networkTransport http.RoundTripper) (fetchbot.Doer, error) {
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
		if afterTime < 1 {
			afterTime = 1
		}
		if j.opts.MaxDelay > j.opts.MinDelay {
			afterTime = int32(<-j.randChan)
			j.log.Printf("Next delay - %v\n", time.Duration(afterTime)*time.Second)
		}
		after := time.After(time.Duration(afterTime) * time.Second)

		httpReq, err := http.NewRequest("GET", j.opts.LoginUrl, nil)
		if err != nil {
			j.log.Println(err)
			return nil, err
		}
		httpReq.Header.Set("User-Agent", j.opts.Useragent)

		httpResp, err := httpClient.Do(httpReq)
		if err != nil {
			j.log.Println(err)
			return nil, err
		}
		pageBody, err := ioutil.ReadAll(httpResp.Body)
		if err != nil {
			j.log.Println("Unable to read http response:", err)
			return nil, err
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
				return nil, err
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
		return nil, err
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
			return nil, err
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
			j.sendPageHTML(nil, phtml)
			return nil, err
		}
		phtml := pb.PageHTML{
			Success:        true,
			Error:          "",
			Sub:            nil, //no subscription object
			Url:            "",
			Httpstatuscode: sc.LoginSuccess,
			Content:        []byte{},
		}
		j.sendPageHTML(nil, phtml)
		j.log.Printf("Logged in. Found '%v' in '%v'\n", j.opts.LoginSuccessCheck.Value, j.opts.LoginSuccessCheck.Key)
	}
	return &Doer{httpClient, j, semaphore.NewWeighted(int64(j.opts.MaxConcurrentRequests)), s}, nil
}

func (s *ideaCrawlerServer) makeHTTPClientRawChrome(j *job) (fetchbot.Doer, error) {
	j.opts.Impolite = true // Always impolite in Chrome mode.
	if j.opts.ChromeBinary == "" {
		j.opts.ChromeBinary = "/usr/lib64/chromium-browser/headless_shell"
	}
	j.cl = chromeclient.NewChromeClient(j.opts.ChromeBinary)
	if j.opts.DomLoadTime > 0 {
		j.cl.SetDomLoadTime(j.opts.DomLoadTime)
	}
	err := j.cl.Start()
	if err != nil {
		j.log.Println("Unable to start chrome:", err)
		return nil, err
	}
	return &Doer{j.cl, j, semaphore.NewWeighted(int64(j.opts.MaxConcurrentRequests)), s}, nil
}

func (s *ideaCrawlerServer) makeHTTPClientLoginChrome(j *job) (fetchbot.Doer, error) {
	j.opts.Impolite = true // Always impolite in Chrome mode.
	if j.opts.ChromeBinary == "" {
		j.opts.ChromeBinary = "/usr/lib64/chromium-browser/headless_shell"
	}
	j.cl = chromeclient.NewChromeClient(j.opts.ChromeBinary)
	if j.opts.DomLoadTime > 0 {
		j.cl.SetDomLoadTime(j.opts.DomLoadTime)
	}
	err := j.cl.Start()
	if err != nil {
		j.log.Println("Unable to start chrome:", err)
		return nil, err
	}
	urlobj, _ := url.Parse(j.opts.LoginUrl)
	req := &http.Request{
		URL: urlobj,
	}
	loginResp, err := j.cl.Do(req)
	if err != nil {
		j.log.Println("Http request to chrome failed:", err)
		return nil, err
	}
	loginBody, err := ioutil.ReadAll(loginResp.Body)
	if err != nil {
		j.log.Println("Unable to read http response", err)
		return nil, err
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
	loggedInResp, err := j.cl.Do(loginJsReq)
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
			return nil, err
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
			j.sendPageHTML(nil, phtml)
			return nil, errors.New(errMsg)
		}
		phtml := pb.PageHTML{
			Success:        true,
			Error:          "",
			Sub:            nil, //no subscription object
			Url:            "",
			Httpstatuscode: sc.LoginSuccess,
			Content:        []byte{},
		}
		j.sendPageHTML(nil, phtml)
		j.log.Printf("Logged in. Found '%v' in '%v'\n", j.opts.LoginSuccessCheck.Value, j.opts.LoginSuccessCheck.Key)
	}
	return &Doer{j.cl, j, semaphore.NewWeighted(int64(j.opts.MaxConcurrentRequests)), s}, nil
}

func (s *ideaCrawlerServer) RunJob(subID string, j *job) {
	err := j.setupJobLogger(subID)
	if err != nil {
		log.Printf("setupJobLogger failed: %s", err)
		return
	}
	j.log.Println("starting job -", subID)
	go j.doneListenersBroadcaster()
	defer func() {
		close(j.subscriber.sendChan)
		j.running = false

		j.done = true
		j.doneChan <- jobDoneSignal{}
		j.log.Println("stopping job -", subID)
	}()
	j.runNumber++

	mux := fetchbot.NewMux()
	mux.HandleErrors(fetchbot.HandlerFunc(j.fetchErrorHandler))
	mux.Response().Method("GET").ContentType("text/html").Handler(fetchbot.HandlerFunc(j.fetchHTTPGetHandler))
	mux.Response().Method("HEAD").ContentType("text/html").Handler(fetchbot.HandlerFunc(j.fetchHTTPHeadHandler))

	f := fetchbot.New(mux)
	j.log.Printf("MaxConcurrentRequests=%v\n", int64(j.opts.MaxConcurrentRequests))

	var networkTransport = j.makeNetworkTransport()

	if j.opts.Chrome == false && j.opts.Login == false {
		f.HttpClient, err = s.makeHTTPClientRawDirect(j, networkTransport)
		if err != nil {
			return
		}
	} else if j.opts.Chrome == false && j.opts.Login == true {
		f.HttpClient, err = s.makeHTTPClientLoginDirect(j, networkTransport)
		if err != nil {
			return
		}
	} else if j.opts.Chrome == true && j.opts.Login == false {
		f.HttpClient, err = s.makeHTTPClientRawChrome(j)
		if err != nil {
			return
		}
		defer j.cl.Stop()
	} else if j.opts.Chrome == true && j.opts.Login == true {
		f.HttpClient, err = s.makeHTTPClientLoginChrome(j)
		if err != nil {
			return
		}
		defer j.cl.Stop()
	}

	f.DisablePoliteness = j.opts.Impolite
	// minimal crawl delay; actual randomized delay is implemented in IdeaCrawlDoer's Do method.
	f.CrawlDelay = 50 * time.Millisecond
	if j.opts.MaxIdleTime == 0 {
		f.WorkerIdleTTL = 600 * time.Second
	} else if j.opts.MaxIdleTime < 10 {
		f.WorkerIdleTTL = 10 * time.Second
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
