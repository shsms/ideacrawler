package main

import "google.golang.org/grpc/connectivity"

type workerQueue []*worker

func (wq *workerQueue) push(w *worker) {
	if w == nil {
		return
	}
	*wq = append(*wq, w)
}

func (wq *workerQueue) pop() *worker {
	q := *wq
	if len(q) == 0 {
		return nil
	}

	var w *worker = q[0]
	*wq = q[1:]
	return w
}

func (wq *workerQueue) next() *worker {
	for {
		w := wq.pop()
		if w == nil {
			return nil
		}

		if w.w.Conn.GetState() == connectivity.Ready {
			wq.push(w)
			return w
		}
	}
}
