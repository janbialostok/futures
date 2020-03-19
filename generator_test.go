package futures

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestNewGenerator(t *testing.T) {
	factory := NewGenerator(func(n interface{}) (interface{}, bool, error) {
		next := n.(int) + 1
		return next, (next < 5), nil
	})

	generator := factory(0)
	next, _, _ := generator.Next()
	assert.Equal(t, 1, next.(int), "should get next value")

	var results []int
	for v := range generator {
		results = append(results, v.Value.(int))
	}

	curr := 2
	for _, v := range results {
		assert.Equal(t, curr, v, "element should match in the expected order")
		curr++
	}

	_, done, _ := generator.Next()
	assert.Equal(t, true, done, "should return true for done value when next is called after generator has yielded its last value")
}