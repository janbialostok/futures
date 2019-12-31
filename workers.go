package main

import (
	"sync"
)

func makeWorker(in chan FutureFunc, out chan Value, kill chan bool) {
	go func() {
	worker:
		for {
			select {
			case _, ok := <-kill:
				if !ok {
					break worker
				}
			default:
			}
			fn := <-in
			if fn != nil {
				result, err := fn()
				out <- Value{result, err}
			}
		}
	}()
}

// WorkerPool defines methods implemented by structs WP and NestedWP. These combined functionalities allow of asynchronous execution of FutureFuncs.
type WorkerPool interface {
	Send(FutureFunc) bool
	Receive() (Value, bool)
	Close() bool
	Fork(int) WorkerPool
}

type WP struct {
	in        chan FutureFunc
	out       Future
	kill      chan bool
	closeLock *sync.Mutex
}

func (w WP) Send(fn FutureFunc) bool {
	select {
	case _, ok := <-w.kill:
		if !ok {
			return false
		}
	default:
	}
	w.in <- fn
	return true
}

func (w WP) Receive() (Value, bool) {
	if v, ok := <-w.out; ok {
		return v, true
	}
	return Value{}, false
}

func (w WP) Close() bool {
	w.closeLock.Lock()
	defer w.closeLock.Unlock()
	select {
	case _, ok := <-w.kill:
		if !ok {
			return false
		}
	default:
	}
	w.kill <- true
	close(w.kill)
	return true
}

func (w WP) Fork(concurrency int) WorkerPool {

}

func NewFuturesWorkerPool(concurrency int) WorkerPool {
	in := make(chan FutureFunc, concurrency)
	out := make(chan Value, concurrency)
	kill := make(chan bool)
	closeChannelLock := sync.Mutex{}

	go func() {
		<-kill
		close(in)
		close(out)
	}()

	for i := 0; i < concurrency; i++ {
		makeWorker(in, out, kill)
	}

	return WP{in, out, kill, &closeChannelLock}
}
