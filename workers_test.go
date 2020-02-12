package futures

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkerPool(t *testing.T) {
	wp := NewFuturesWorkerPool(2)

	assert.Equal(t, true, wp.Send(func() (interface{}, error) {
		return "foobar", nil
	}), "should be able to execute an FutureFunc async")

	value, _ := wp.Receive()
	assert.Equal(t, "foobar", value.Data.(string), "should receive the returned value of the FutureFunc")

	for i := 0; i < 4; i++ {
		go func(index int) {
			wp.Send(func() (interface{}, error) {
				return index, nil
			})
		}(i)
	}

	var expected []int
	for {
		value, _ := wp.Receive()
		expected = append(expected, value.Data.(int))
		if len(expected) == 4 {
			break
		}
	}

	assert.ElementsMatch(t, []int{0, 1, 2, 3}, expected, "should execute any number of FutureFuncs and receive the correct return values")

	out := make(chan Value, 1)
	assert.Equal(t, true, wp.Do(out, func() (interface{}, error) {
		return "hello world", nil
	}), "should execute FutureFunc in worker go routine and return true")
	value = <-out
	assert.Equal(t, "hello world", value.Data.(string), "should receive returned value from FutureFunc in specified out channel")

	assert.Equal(t, true, wp.Close(), "should return true the first time Close is called")

	assert.Equal(t, false, wp.Send(func() (interface{}, error) {
		return "foobar", nil
	}), "should return false if Send is called and workers have been closed")

	_, ok := wp.Receive()
	assert.Equal(t, false, ok, "should return false if Receive is called and workers have been closed")

	assert.Equal(t, false, wp.Do(out, func() (interface{}, error) {
		return "hello world", nil
	}), "should return false if workers have been closed")

	assert.Equal(t, false, wp.Close(), "should return false if Close is called after workers have alreay been closed")
}

func TestNestedWorkerPool(t *testing.T) {
	wp := NewFuturesWorkerPool(4)
	np := wp.Fork(2)

	assert.Equal(t, true, np.Send(func() (interface{}, error) {
		return "foobar", nil
	}), "should be able to execute an FutureFunc async")

	value, _ := np.Receive()
	assert.Equal(t, "foobar", value.Data.(string), "should receive the returned value of the FutureFunc")

	out := make(chan Value, 1)
	assert.Equal(t, true, np.Do(out, func() (interface{}, error) {
		return "hello world", nil
	}), "should execute FutureFunc in worker go routine and return true")
	value = <-out
	assert.Equal(t, "hello world", value.Data.(string), "should receive returned value from FutureFunc in specified out channel")

	assert.Equal(t, true, np.Close(), "should return true the first time Close is called")

	assert.Equal(t, false, np.Send(func() (interface{}, error) {
		return "foobar", nil
	}), "should return false if Send is called and workers have been closed")

	_, ok := np.Receive()
	assert.Equal(t, false, ok, "should return false if Receive is called and workers have been closed")

	assert.Equal(t, false, np.Do(out, func() (interface{}, error) {
		return "hello world", nil
	}), "should return false if workers have been closed")

	assert.Equal(t, false, np.Close(), "should return false if Close is called after workers have alreay been closed")

	assert.Equal(t, true, wp.Send(func() (interface{}, error) {
		return "foobar", nil
	}), "should be able to execute an FutureFunc async with parent worker pool after nested pool is closed")

	value, _ = wp.Receive()
	assert.Equal(t, "foobar", value.Data.(string), "should receive the returned value of the FutureFunc with parent worker pool after nested pool is closed")

	np = wp.Fork(2)
	wp.Close()

	assert.Equal(t, false, np.Send(func() (interface{}, error) {
		return "foobar", nil
	}), "should return false if Send is called and parent workers have been closed")

	_, ok = np.Receive()
	assert.Equal(t, false, ok, "should return false if Receive is called and parent workers have been closed")

	assert.Equal(t, false, np.Do(out, func() (interface{}, error) {
		return "hello world", nil
	}), "should return false if parent workers have been closed")

	assert.Equal(t, false, np.Close(), "should return false if Close is called after parents workers have alreay been closed")
}
