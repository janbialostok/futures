package futures

import (
	"sync"
)

type skipOutChannel struct{}

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
				if _, ok := result.(skipOutChannel); !ok {
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
	Do(chan Value, FutureFunc) bool
}

// WorkerPool manages go routines used for executing FutureFuncs
type WorkerPool struct {
	in        chan FutureFunc
	out       Future
	kill      chan bool
	closeLock *sync.Mutex
	sendLock  *sync.Mutex
}

// Send pushes a FutureFunc to a channel that worker go routines poll and execute from
func (w WorkerPool) Send(fn FutureFunc) bool {
	w.sendLock.Lock()
	defer w.sendLock.Unlock()
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

// Receive listens on the out channel and waits for a value to be returned from a FutureFunc execution
func (w WorkerPool) Receive() (Value, bool) {
	if v, ok := <-w.out; ok {
		return v, true
	}
	return Value{}, false
}

// Close kills all worker go routines and closes in and out channels
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
	close(w.kill)
	return true
}

// Fork creates a NestedWorkerPool from parent WorkerPool which shares an in channel and worker go routines
func (w WorkerPool) Fork(concurrency int) WorkerPoolInterface {
	out := make(chan Value, concurrency)
	kill := make(chan bool)
	go func() {
		select {
		case _, ok := <-out:
			if !ok {
				return
			}
		default:
		}
		select {
		case <-kill:
			close(out)
		case <-w.kill:
			close(out)
		}
	}()
	return NestedWorkerPool{
		WorkerPool: WorkerPool{
			in:        w.in,
			kill:      w.kill,
			closeLock: w.closeLock,
			sendLock:  w.sendLock,
		},
		out:  out,
		kill: kill,
	}
}

// Do executes a FutureFunc in a worker go routine but writes the returned value to the specified out channel
func (w WorkerPool) Do(out chan Value, fn FutureFunc) bool {
	return w.Send(func() (interface{}, error) {
		result, err := fn()
		out <- Value{result, err}
		return skipOutChannel{}, nil
	})
}

// NestedWorkerPool shares resources with a parent WorkerPool but can be indepedently closed and only receives values directly sent to it
type NestedWorkerPool struct {
	WorkerPool
	out  chan Value
	kill chan bool
}

// Send pushes a FutureFunc to a parent WorkerPool but writes the result to the nested out channel
func (n NestedWorkerPool) Send(fn FutureFunc) bool {
	select {
	case _, ok := <-n.kill:
		if !ok {
			return false
		}
	default:
	}
	return n.WorkerPool.Send(func() (interface{}, error) {
		result, err := fn()
		select {
		case _, ok := <-n.kill:
			if !ok {
				return skipOutChannel{}, nil
			}
		default:
		}
		n.out <- Value{result, err}
		return skipOutChannel{}, nil
	})
}

// Receive listens on the out channel and waits for a value to be returned from a FutureFunc execution
func (n NestedWorkerPool) Receive() (Value, bool) {
	if v, ok := <-n.out; ok {
		return v, true
	}
	return Value{}, false
}

// Close closes nested out channel and blocks any subsequent writes to the parent in channel. Closing will not effect parent WorkerPool resources
func (n NestedWorkerPool) Close() bool {
	n.closeLock.Lock()
	defer n.closeLock.Unlock()
	select {
	case _, ok := <-n.WorkerPool.kill:
		if !ok {
			return false
		}
	default:
	}
	select {
	case _, ok := <-n.kill:
		if !ok {
			return false
		}
	default:
	}
	close(n.kill)
	return true
}

// Do executes a FutureFunc in a parent worker go routine but writes the returned value to the specified out channel
func (n NestedWorkerPool) Do(out chan Value, fn FutureFunc) bool {
	select {
	case _, ok := <-n.kill:
		if !ok {
			return false
		}
	default:
	}
	return n.WorkerPool.Do(out, fn)
}

// NewFuturesWorkerPool creates a WorkerPool with the specified number of workers as define by the concurrency argument
func NewFuturesWorkerPool(concurrency int) WorkerPoolInterface {
	in := make(chan FutureFunc, concurrency)
	out := make(chan Value, concurrency)
	kill := make(chan bool)
	closeChannelLock := sync.Mutex{}
	sendLock := sync.Mutex{}

	go func() {
		<-kill
		sendLock.Lock()
		defer sendLock.Unlock()
		close(in)
		close(out)
	}()

	for i := 0; i < concurrency; i++ {
		makeWorker(in, out, kill)
	}

	return WorkerPool{in, out, kill, &closeChannelLock, &sendLock}
}
