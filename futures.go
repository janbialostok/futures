package futures

// FutureFunc specifies the function signature expected for a Future
type FutureFunc func() (interface{}, error)

// ThenableFunc specifies the function signature expected for a Thenable
type ThenableFunc func(interface{}) (interface{}, error)

// CatchableFunc specifies the function signature expected for a Catchable
type CatchableFunc func(error) (interface{}, error)

// ResolvedFutureError implements the error interface and returns a standard error for attempting to chain from a Future that has already been resolved
type ResolvedFutureError struct{}

// Error returns an error message for ResolvedFutureError
func (ResolvedFutureError) Error() string {
	return "future value has already been resolved"
}

// Value contains the resolved value of a Future or the resulting error
type Value struct {
	Data  interface{}
	Error error
}

// Future is a read-only channel that is meant to be read from only once and has the resulting value of a FutureFunc
type Future <-chan Value

func (f Future) resolveLast() (Value, error) {
	prev, ok := <-f
	if !ok {
		return prev, ResolvedFutureError{}
	}
	return prev, nil
}

// Then uses function composition to execute a ThenableFunc in the order in which it was defined if there was no prior error returned with the result of the previous function
func (f Future) Then(fn ThenableFunc) Future {
	return NewFuture(func() (interface{}, error) {
		prev, err := f.resolveLast()
		if err != nil {
			return nil, err
		}
		if prev.Error != nil {
			return nil, prev.Error
		}
		return fn(prev.Data)
	})
}

// Catch uses function composition to execute a CatchableFunc in the order in which it was defined if there was a prior error returned with the error that was returned from the prior function
func (f Future) Catch(fn CatchableFunc) Future {
	return NewFuture(func() (interface{}, error) {
		prev, err := f.resolveLast()
		if err != nil {
			return nil, err
		}
		if prev.Error == nil {
			return prev.Data, nil
		}
		return fn(prev.Error)
	})
}

// Finally uses function composition to execute a FutureFunc in the order in which it was defined regardless of the result of prior functions
func (f Future) Finally(fn FutureFunc) Future {
	return NewFuture(func() (interface{}, error) {
		if _, err := f.resolveLast(); err != nil {
			return nil, err
		}
		return fn()
	})
}

// NewFuture returns a Future that propagates a Value containing the result of the execution of the defined FutureFunc argument
func NewFuture(fn FutureFunc) Future {
	c := make(chan Value, 1)
	go func() {
		v := Value{}
		v.Data, v.Error = fn()
		c <- v
		close(c)
	}()
	return c
}

// Series executes ThenableFunc's in the order in which they appear in the argument slice with the argument for the first ThenableFunc being the first argument passed to Series
func Series(argv interface{}, fns ...ThenableFunc) Future {
	f := NewFuture(func() (interface{}, error) {
		return argv, nil
	})
	for _, fn := range fns {
		f = f.Then(fn)
	}
	return f
}

// Pipe returns a function that executes the defined argument slice with Series
func Pipe(fns ...ThenableFunc) func(interface{}) Future {
	return func(argv interface{}) Future {
		return Series(argv, fns...)
	}
}

func resolveSliceValuesFromWorkerPool(length int, wp WorkerPoolInterface) Future {
	return NewFuture(func() (interface{}, error) {
		defer wp.Close()
		index := 0
		result := make([]interface{}, length)
		for {
			if v, ok := wp.Receive(); ok && v.Error == nil {
				result[index] = v.Data
				index++
				if index == length {
					break
				}
			} else {
				return nil, v.Error
			}
		}
		return result, nil
	})
}

// AllWithWorkerPool returns a Future which will resolve with all the values passed in the values argument. FutureFunc's and Futures passed in the argument are executed or resolved and all other values are returned as is. The provided WorkerPool is forked and closed at the end of execution.
func AllWithWorkerPool(values []interface{}, concurrency int, wp WorkerPoolInterface) Future {
	np := wp.Fork(concurrency)
	for _, v := range values {
		switch f := v.(type) {
		case Future:
			np.Send(func() (interface{}, error) {
				value := <-f
				return value.Data, value.Error
			})
			break
		case FutureFunc:
			np.Send(f)
			break
		case func() (interface{}, error):
			np.Send(f)
			break
		default:
			np.Send(func() (interface{}, error) {
				return f, nil
			})
		}
	}
	return resolveSliceValuesFromWorkerPool(len(values), np)
}

// All calls AllWithWorkerPool but first creates a new WorkerPool with the specified concurrency
func All(values []interface{}, concurrency int) Future {
	wp := NewFuturesWorkerPool(concurrency)
	return AllWithWorkerPool(values, concurrency, wp)
}

// MapWithWorkerPool calls the defined fn Thenabled argument with each of the values provided in the values arugment and returns a Future that will resolve with the resulting values. The provided WorkerPool is forked and closed at the end of execution.
func MapWithWorkerPool(values []interface{}, fn ThenableFunc, concurrency int, wp WorkerPoolInterface) Future {
	np := wp.Fork(concurrency)
	for _, v := range values {
		curr := v
		np.Send(func() (interface{}, error) {
			return fn(curr)
		})
	}
	return resolveSliceValuesFromWorkerPool(len(values), np)
}

// Map calls MapWithWorkerPool but first creates a WorkerPool with the specified concurrency
func Map(values []interface{}, fn ThenableFunc, concurrency int) Future {
	wp := NewFuturesWorkerPool(concurrency)
	return MapWithWorkerPool(values, fn, concurrency, wp)
}
