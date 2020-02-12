package futures

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFuture(t *testing.T) {
	f := NewFuture(func() (interface{}, error) {
		return "foobar", nil
	})

	value := <-f
	assert.Equal(t, "foobar", value.Data.(string), "should resolve the correct value from Future")

	f = NewFuture(func() (interface{}, error) {
		return nil, fmt.Errorf("some error")
	})

	value = <-f
	assert.Error(t, value.Error, "should resolve Future with an error")

	value = <-f.Then(func(value interface{}) (interface{}, error) {
		return nil, nil
	})

	assert.Equal(t, ResolvedFutureError{}.Error(), value.Error.Error(), "should return a resolved future error if Future is reused being resolved")
}

func TestThen(t *testing.T) {
	f := NewFuture(func() (interface{}, error) {
		return 1, nil
	}).
		Then(func(value interface{}) (interface{}, error) {
			return value.(int) + 1, nil
		})

	value := <-f
	assert.Equal(t, 2, value.Data.(int), "should resolve value after executing ThenableFunc")
}

func TestCatch(t *testing.T) {
	var thenDidExecute bool
	f := NewFuture(func() (interface{}, error) {
		return nil, fmt.Errorf("some error")
	}).
		Then(func(value interface{}) (interface{}, error) {
			thenDidExecute = true
			return value.(int) + 1, nil
		}).
		Catch(func(err error) (interface{}, error) {
			return 0, nil
		})

	value := <-f
	assert.Equal(t, 0, value.Data.(int), "should resolve value after executing CatchableFunc")

	assert.Equal(t, false, thenDidExecute, "should not execute ThenableFunc if error is returned in preceding FutureFunc")
}

func TestFinally(t *testing.T) {
	f := NewFuture(func() (interface{}, error) {
		return 1, nil
	}).
		Then(func(value interface{}) (interface{}, error) {
			return value.(int) + 1, nil
		}).
		Catch(func(err error) (interface{}, error) {
			return 0, nil
		}).
		Finally(func() (interface{}, error) {
			return "done", nil
		})

	value := <-f
	assert.Equal(t, "done", value.Data.(string), "should execute Finally FutureFunc")

	f = NewFuture(func() (interface{}, error) {
		return nil, fmt.Errorf("some error")
	}).
		Then(func(value interface{}) (interface{}, error) {
			return value.(int) + 1, nil
		}).
		Catch(func(err error) (interface{}, error) {
			return nil, fmt.Errorf("another error")
		}).
		Finally(func() (interface{}, error) {
			return "done", nil
		})

	value = <-f
	assert.Equal(t, "done", value.Data.(string), "should execute Finally FutureFunc when there is an error")
}

func TestSeries(t *testing.T) {
	var fns []ThenableFunc
	for i := 0; i < 3; i++ {
		fns = append(fns, func(value interface{}) (interface{}, error) {
			return value.(int) + 1, nil
		})
	}
	f := Series(0, fns...)
	value := <-f
	assert.Equal(t, 3, value.Data.(int), "should call each of the ThenableFunc's in the Series sequentially")
}

func TestPipe(t *testing.T) {
	var fns []ThenableFunc
	for i := 0; i < 3; i++ {
		fns = append(fns, func(value interface{}) (interface{}, error) {
			return value.(int) + 1, nil
		})
	}
	pipe := Pipe(fns...)
	value := <-pipe(0)
	assert.Equal(t, 3, value.Data.(int), "should call each of the ThenableFunc's in the Pipe sequentially with the correct first arg")

	value = <-pipe(1)
	assert.Equal(t, 4, value.Data.(int), "should be able to call method returned by Pipe multiple times with different first args")
}

func TestAll(t *testing.T) {
	values := []interface{}{
		FutureFunc(func() (interface{}, error) {
			return 0, nil
		}),
		NewFuture(func() (interface{}, error) {
			return 1, nil
		}),
		2,
		func() (interface{}, error) {
			return 3, nil
		},
	}
	f := All(values, 3)
	result := <-f
	assert.ElementsMatch(t, []interface{}{0, 1, 2, 3}, result.Data.([]interface{}), "should resolve all values in slice")

	values = []interface{}{
		NewFuture(func() (interface{}, error) {
			return 0, nil
		}),
		func() (interface{}, error) {
			return nil, fmt.Errorf("some error")
		},
	}

	result = <-All(values, 3)
	assert.Error(t, result.Error, "should resolve with an error if any Future's or FutureFunc's return an error")

	result = <-All([]interface{}{}, 3)
	assert.Empty(t, result.Data.([]interface{}), "should handle empty input values")
}

func TestMap(t *testing.T) {
	result := <-Map([]interface{}{
		1,
		2,
		3,
	}, func(value interface{}) (interface{}, error) {
		return value.(int) * 2, nil
	}, 2)

	assert.ElementsMatch(t, []interface{}{2, 4, 6}, result.Data.([]interface{}), "should map over values with ThenableFunc")

	result = <-Map([]interface{}{"foobar"}, func(value interface{}) (interface{}, error) {
		return nil, fmt.Errorf("some error")
	}, 1)
	assert.Error(t, result.Error, "should resolve with an error if any Future's or FutureFunc's return an error")
}
