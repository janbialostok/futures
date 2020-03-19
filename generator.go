package futures

// GeneratorFunc specifies the function signature for each step of the generator execution.
// Returns an interface result, boolean done status and error value with the result of the function being passed as the argument to the next invocation.
type GeneratorFunc func(interface{}) (interface{}, bool, error)

// GeneratorValue contains the output of a GeneratorFunc execution
type GeneratorValue struct {
	done  bool
	Value interface{}
	Error error
}

// Generator is a read-only channel the receives the results of the stepwise execution of GeneratorFunc's
type Generator <-chan GeneratorValue

// Next yields the result of the current step of the GeneratorFunc's execution
func (gen Generator) Next() (interface{}, bool, error) {
	if next, ok := <-gen; ok {
		return next.Value, next.done, next.Error
	}
	return nil, true, nil
}

// NewGenerator returns a factory method for creating Generator's.
// The argument of the factory function is passed as the initial input to the GeneratorFunc.
func NewGenerator(fn GeneratorFunc) func(interface{}) Generator {
	return func(argv interface{}) Generator {
		out := make(chan GeneratorValue, 1)
		go func() {
			input := argv
			for {
				result, done, err := fn(input)
				input = result
				out <- GeneratorValue{done, result, err}
				if done {
					close(out)
					return
				}
			}
		}()
		return out
	}
}
