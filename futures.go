package main

type FutureFunc func() (interface{}, error)
type ThenableFunc func(interface{}) (interface{}, error)
type CatchableFunc func(error) (interface{}, error)

type ResolvedFutureError struct{}

func (ResolvedFutureError) Error() string {
	return "future value has already been resolved"
}

type Value struct {
	Data  interface{}
	Error error
}

type Future <-chan Value

func (this Future) resolveLast() (Value, error) {
	prev, ok := <-this
	if !ok {
		return prev, ResolvedFutureError{}
	}
	return prev, nil
}

func (this Future) Then(fn ThenableFunc) Future {
	return NewFuture(func() (interface{}, error) {
		prev, err := this.resolveLast()
		if err != nil {
			return nil, err
		}
		if prev.Error != nil {
			return nil, prev.Error
		}
		return fn(prev.Data)
	})
}

func (this Future) Catch(fn CatchableFunc) Future {
	return NewFuture(func() (interface{}, error) {
		prev, err := this.resolveLast()
		if err != nil {
			return nil, err
		}
		if prev.Error == nil {
			return prev.Data, nil
		}
		return fn(prev.Error)
	})
}

func (this Future) Finally(fn FutureFunc) Future {
	return NewFuture(func() (interface{}, error) {
		if _, err := this.resolveLast(); err != nil {
			return nil, err
		}
		return fn()
	})
}

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

func Series(argv interface{}, fns ...ThenableFunc) Future {
	f := NewFuture(func() (interface{}, error) {
		return argv, nil
	})
	for _, fn := range fns {
		f = f.Then(fn)
	}
	return f
}

func Pipe(fns ...ThenableFunc) func(interface{}) Future {
	return func(argv interface{}) Future {
		return Series(argv, fns...)
	}
}

func resolveSliceValuesFromWorkerPool(length int, wp WorkerPoolInterface) Future {
	return NewFuture(func() (interface{}, error) {
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
		default:
			np.Send(func() (interface{}, error) {
				return f, nil
			})
		}
	}
	return resolveSliceValuesFromWorkerPool(len(values), np)
}

func All(values []interface{}, concurrency int) Future {
	wp := NewFuturesWorkerPool(concurrency)
	return AllWithWorkerPool(values, concurrency, wp)
}

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

func Map(values []interface{}, fn ThenableFunc, concurrency int) Future {
	wp := NewFuturesWorkerPool(concurrency)
	return MapWithWorkerPool(values, fn, concurrency, wp)
}
