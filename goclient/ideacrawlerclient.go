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

package goclient

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	pb "github.com/ideas2it/ideacrawler/protofiles"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type PageHTML = pb.PageHTML

type Worker struct {
	Conn   *grpc.ClientConn
	Client pb.IdeaCrawlerClient
}

type CrawlJob struct {
	dopt *pb.DomainOpt

	startedChan    chan struct{} // if this is closed, job has started.
	stoppedChan    chan struct{} // if this is closed, job has stopped.
	addPagesClient pb.IdeaCrawler_AddPagesClient
	sub            *pb.Subscription

	Worker              *Worker
	callback            func(*PageHTML, *CrawlJob)
	usePageChan         bool
	PageChan            <-chan *pb.PageHTML
	implPageChan        chan *pb.PageHTML
	useAnalyzedURLChan  bool
	implAnalyzedURLChan chan *pb.UrlList
	AnalyzedURLChan     <-chan *pb.UrlList
	callbackSeedURL     bool

	cleanUpFunc func()
}

type KVMap = map[string]string

const (
	PageReqType_BUILTINJS = pb.PageReqType_BUILTINJS
	PageReqType_JSCRIPT   = pb.PageReqType_JSCRIPT
)

func NewWorker(socket string, tcpconn net.Conn) (*Worker, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	if tcpconn != nil {
		opts = append(opts, grpc.WithDialer(func(string, time.Duration) (net.Conn, error) {
			return tcpconn, nil
		}))
		opts = append(opts, grpc.WithDisableRetry())
	}
	conn, err := grpc.Dial(socket, opts...)
	if err != nil {
		return nil, err
	}
	client := pb.NewIdeaCrawlerClient(conn)
	return &Worker{conn, client}, nil
}

func (w *Worker) Close() {
	w.Conn.Close()
}

func (w *Worker) NewCrawlJob(opts ...Option) *CrawlJob {
	dopt := &pb.DomainOpt{
		MinDelay:              5,
		Depth:                 -1,
		DomLoadTime:           5,
		Useragent:             "Fetchbot",
		MaxConcurrentRequests: 5,
	}
	var cj = &CrawlJob{
		dopt:        dopt,
		Worker:      w,
		startedChan: make(chan struct{}),
		stoppedChan: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(cj)
	}

	return cj
}

func (cj *CrawlJob) initAnalyzedURL() {
	if cj.useAnalyzedURLChan == false {
		return
	}
	if cj.sub == nil {
		log.Println("No job subscription. SetAnalyzedURLs failed.")
		return
	}
	urlstream, err := cj.Worker.Client.GetAnalyzedURLs(context.Background(), cj.sub, grpc.MaxCallRecvMsgSize((2*1024*1024*1024)-1))
	if err != nil {
		log.Println("Box is possibly down. SetAnalyzedURLs failed:", err)
		return
	}
	go cj.listenAnalyzedURLs(urlstream)
}

func (cj *CrawlJob) listenAnalyzedURLs(urlstream pb.IdeaCrawler_GetAnalyzedURLsClient) {
	for {
		urlList, err := urlstream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
			break
		}
		cj.implAnalyzedURLChan <- urlList
	}
}

func (cj *CrawlJob) AddPage(url, metaStr string) error {
	if cj.IsAlive() == false {
		errorMsg := "AddPage function can't be called when crawl job is not running. Please start crawl job first then call Addpage."
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}
	if cj.addPagesClient == nil {
		var err error
		cj.addPagesClient, err = cj.Worker.Client.AddPages(context.Background())
		if err != nil {
			cj.addPagesClient = nil
			return err
		}
	}
	return cj.addPagesClient.Send(&pb.PageRequest{
		Sub:     cj.sub,
		Reqtype: pb.PageReqType_GET,
		Url:     url,
		MetaStr: metaStr,
	})
}

func (cj *CrawlJob) AddJS(typ pb.PageReqType, url, js, metaStr string) error {
	if cj.IsAlive() == false {
		errorMsg := "AddJS function can't be called when crawl job is not running. Please start crawl job first then call AddJS."
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}
	if cj.addPagesClient == nil {
		var err error
		cj.addPagesClient, err = cj.Worker.Client.AddPages(context.Background())
		if err != nil {
			cj.addPagesClient = nil
			return err
		}
	}
	return cj.addPagesClient.Send(&pb.PageRequest{
		Sub:     cj.sub,
		Reqtype: typ,
		Url:     url,
		Js:      js,
		MetaStr: metaStr,
	})
}

func (cj *CrawlJob) Start() {
	go cj.Run()
	time.Sleep(2 * time.Second)
}

func (cj *CrawlJob) IsAlive() bool {
	select {
	case <-cj.startedChan:
	default: // startedChan is still open, so job has not started
		return false
	}
	select {
	case _, ok := <-cj.stoppedChan:
		if ok == false {
			return false
		}
	default: // stoppedChan is still open, so job is running
		return true
	}
	return false // never reached
}

func (cj *CrawlJob) Stop() {
	if cj.IsAlive() {
		cj.Worker.Client.CancelJob(context.Background(), cj.sub)
	}
}

func (cj *CrawlJob) Run() {
	close(cj.startedChan)
	defer func() {
		close(cj.stoppedChan)
		if cj.cleanUpFunc != nil {
			cj.cleanUpFunc()
		}
	}()

	if cj.usePageChan == true && cj.callback != nil {
		log.Fatal("Callback channel and function both can't be used at the same time")
	} else if cj.usePageChan == false && cj.callback == nil {
		log.Fatal("Please set pageChan to get callbacks on,  or provide a callback function")
	}

	pagestream, err := cj.Worker.Client.AddDomainAndListen(context.Background(), cj.dopt, grpc.MaxCallRecvMsgSize((2*1024*1024*1024)-1))
	if err != nil {
		log.Println("Box is possibly down. AddDomainAndListen failed:", err)
		return
	}
	subpage, err := pagestream.Recv()
	if err == io.EOF {
		return
	}
	if err != nil {
		fmt.Println(err)
		return
	}
	cj.sub = subpage.Sub
	cj.initAnalyzedURL()
	phChan := make(chan *pb.PageHTML, 1000)
	defer close(phChan)

	go func() {
		time.Sleep(3 * time.Second) // This is to make sure callbacks don't start until Start() function exits.  Start sleep for 2 seconds.
		if cj.usePageChan {
			for ph := range phChan {
				cj.implPageChan <- ph
			}
			close(cj.implPageChan)
		} else {
			for ph := range phChan {
				cj.callback(ph, cj)
			}
		}
	}()

	for {
		page, err := pagestream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
			break
		}
		phChan <- page
	}
}

func NewPageChan() chan *PageHTML {
	return make(chan *PageHTML, 100)
}

func NewAnalyzedURLChan() chan *pb.UrlList {
	return make(chan *pb.UrlList, 100)
}
