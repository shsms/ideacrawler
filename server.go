package main

import (
	"fmt"
	"log"
	"math"
	"net"
	"os"

	"github.com/hashicorp/yamux"
	gc "github.com/shsms/ideacrawler/goclient"
	"google.golang.org/grpc/connectivity"
)

type worker struct {
	id   string
	w    *gc.Worker
	job  *gc.CrawlJob // raw job
	cJob *gc.CrawlJob // chrome job
}

type workerManager struct {
	workers     map[string]*worker
	addChan     chan *worker // add new worker
	nextReqChan chan struct{}
	nextChan    chan *worker // get next worker
	pageChan    chan *gc.PageHTML
}

func newWorkerManager() *workerManager {
	wm := &workerManager{
		workers:     make(map[string]*worker),
		addChan:     make(chan *worker),
		nextReqChan: make(chan struct{}),
		nextChan:    make(chan *worker),
		pageChan:    gc.NewPageChan(),
	}
	return wm
}

func (wm *workerManager) addWorker(w *worker) {
	wm.addChan <- w
}

func (wm *workerManager) nextWorker() *worker {
	wm.nextReqChan <- struct{}{}
	return <-wm.nextChan
}

func (wm *workerManager) run() {
	for {
	wmNextCycle:
		select {
		case w := <-wm.addChan:
			// TODO: check worker already exists
			wm.workers[w.id] = w
		case <-wm.nextReqChan:
			// TODO: switch from below round-robin
			// approach to a load based approach.
			for id, w := range wm.workers {
				if w.w.Conn.GetState() != connectivity.Ready {
					delete(wm.workers, id)
					continue
				}
				wm.nextChan <- w
				break wmNextCycle
			}
			log.Println("No active workers found")
			wm.nextChan <- nil
		}
	}
}

func startCrawlerServer() {
	wm := newWorkerManager()
	wm.run()
	wm.startClusterListener()
}

func (wm *workerManager) startClusterListener() {
	listener, err := net.Listen("tcp", cliParams.ClusterListenAddress)
	if err != nil {
		log.Printf("failed to listen: %#v", err)
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Println("Listening on", cliParams.ClusterListenAddress)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("failed to accept: %#v", err)
			continue
		}
		go wm.createNewClient(conn)
	}
}

func (wm *workerManager) createNewClient(conn net.Conn) {
	session, err := yamux.Client(conn, nil)
	if err != nil {
		log.Println("failed to create yamux client:", err)
		return
	}
	// Open a new stream
	yconn, err := session.Open()
	if err != nil {
		log.Println("failed to open yamux conn:", err)
		return
	}
	w, err := gc.NewWorker("", yconn)
	if err != nil {
		log.Println("failed to create gc.Worker:", err)
		return
	}
	j := newWorkerJob(w, false, wm.pageChan)
	if j == nil {
		return
	}
	id, err := w.GetWorkerID()
	if err != nil {
		log.Println("failed to get workerID:", err)
		return
	}
	// TODO: grpc getWorkerID
	wm.addWorker(&worker{
		id:  id,
		w:   w,
		job: j,
	})
}

func newWorkerJob(w *gc.Worker, chrome bool, pageChan chan *gc.PageHTML) *gc.CrawlJob {
	opts := []gc.Option{
		gc.NoFollow(),
		gc.Impolite(),
		gc.PageChan(pageChan),
		gc.MaxIdleTime(math.MaxInt64),
	}

	if chrome == true {
		opts = append(opts, gc.Chrome(true, "/usr/bin/chromium"))
	}
	z, err := gc.NewCrawlJob(w, gc.NewJobSpec(opts...))
	if err != nil {
		log.Println("failed to create CrawlJob:", err)
		return nil
	}
	return z

}
