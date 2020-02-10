package main

import (
	"fmt"
	"time"
)

func main() {
	f := Pipe(
		func(_ interface{}) (interface{}, error) {
			time.Sleep(1 * time.Second)
			return 1, nil
		},
		func(argv interface{}) (interface{}, error) {
			time.Sleep(1 * time.Second)
			fmt.Println(fmt.Sprintf("argv %#v", argv))
			v, ok := argv.(int)
			if !ok {
				return nil, fmt.Errorf("received argument of wrong type")
			}
			return v + 2, nil
		},
	)

	fmt.Println("logging this")
	result := <-f(nil)

	fmt.Println(fmt.Sprintf("from then %#v", result))

	// wp := NewFuturesWorkerPool(9)
	// numops := 10
	// for i := 0; i < numops; i++ {
	// 	go func(index int) {
	// 		wp.Send(func() (interface{}, error) {
	// 			time.Sleep(1 * time.Second)
	// 			return index, nil
	// 		})
	// 	}(i)
	// }

	// for i := 0; i < numops; i++ {
	// 	go func(index int) {
	// 		wp.Send(func() (interface{}, error) {
	// 			time.Sleep(1 * time.Second)
	// 			return index, nil
	// 		})
	// 	}(i)
	// }

	// count := 0
	// for {
	// 	v, ok := wp.Receive()
	// 	if !ok {
	// 		break
	// 	}
	// 	fmt.Println(fmt.Sprintf("%#v", v))
	// 	count++
	// 	if count == 5 {
	// 		wp.Close()
	// 	}
	// }

	a := Map([]interface{}{
		1,
		2,
		3,
	}, func(v interface{}) (interface{}, error) {
		time.Sleep(1 * time.Second)
		return v.(int) * 2, nil
	}, 1)

	v := <-a
	fmt.Println(fmt.Sprintf("%#v", v.Data))
}
