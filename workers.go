package main

import (
	"sync"
)

type skipProduceValue struct{}

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
				if _, ok := result.(skipProduceValue); !ok {
					out <- Value{result, err}
				}
			}
		}
	}()
}

// WorkerPoolInterface defines methods implemented by structs WorkerPool and NestedWorkerPool. These combined functionalities allow of asynchronous execution of FutureFuncs.
type WorkerPoolInterface interface {
	Send(FutureFunc) bool
	Receive() (Value, bool)
	Close() bool
	Fork(int) WorkerPoolInterface
}

type WorkerPool struct {
	in        chan FutureFunc
	out       Future
	kill      chan bool
	closeLock *sync.Mutex
}

func (w WorkerPool) Send(fn FutureFunc) bool {
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

func (w WorkerPool) Receive() (Value, bool) {
	if v, ok := <-w.out; ok {
		return v, true
	}
	return Value{}, false
}

func (w WorkerPool) Close() bool {
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

func (w WorkerPool) Fork(concurrency int) WorkerPoolInterface {
	out := make(chan Value, concurrency)
	return NestedWorkerPool{
		WorkerPool: WorkerPool{
			in:        w.in,
			kill:      w.kill,
			closeLock: w.closeLock,
		},
		out: out,
	}
}

type NestedWorkerPool struct {
	WorkerPool
	out chan Value
}

func (n NestedWorkerPool) Send(fn FutureFunc) bool {
	return n.WorkerPool.Send(func() (interface{}, error) {
		result, err := fn()
		n.out <- Value{result, err}
		return skipProduceValue{}, err
	})
}

func (n NestedWorkerPool) Receive() (Value, bool) {
	if v, ok := <-n.out; ok {
		return v, true
	}
	return Value{}, false
}

func NewFuturesWorkerPool(concurrency int) WorkerPoolInterface {
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

	return WorkerPool{in, out, kill, &closeChannelLock}
}
