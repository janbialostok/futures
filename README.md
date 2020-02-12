## Install

```sh
$ go get github.com/janbialostok/futures
```

## Future Usage

Futures are used to asyncronously execute a function

```go
package main

import (
  "github.com/janbialostok/futures"
  "net/http"
)

func main() {
  f := futures.NewFuture(func() (interface{}, error) {
    return http.Get("website.com")
  })

  // do some other stuff

  if value := <-f; value.Error != nil {
    // handle error
  } else {
    result, ok := value.Data.(*http.Response)
  }
}
```

With futures.Future.Then you can chain behavior

```go
package main

import (
  "github.com/janbialostok/futures"
  "net/http"
  "encoding/json"
  "fmt"
)

func main() {
  f := futures.NewFuture(func() (interface{}, error) {
    return http.Get("website.com")
  }).
    Then(func(value interface{}) (interface{}, error) {
      result, ok := value.Data.(*http.Response)
      if ok {
        var payload map[string]interface{}
        err := json.NewDecoder(result).Decode(&payload)
        return payload, err
      }
      return nil, fmt.Errorf("could not read response")
    })

  // do some other stuff

  if value := <-f; value.Error != nil {
    // handle error
  } else {
    result, ok := value.Data.(map[string]interface{})
  }
}
```

With futures.Future.Catch you can handle errors returned during execution

```go
package main

import (
  "github.com/janbialostok/futures"
  "net/http"
  "encoding/json"
  "fmt"
)

func main() {
  f := futures.NewFuture(func() (interface{}, error) {
    return http.Get("website.com")
  }).
    Then(func(value interface{}) (interface{}, error) {
      result, ok := value.Data.(*http.Response)
      if ok {
        var payload map[string]interface{}
        err := json.NewDecoder(result).Decode(&payload)
        return payload, err
      }
      return nil, fmt.Errorf("could not read response")
    }).
    Catch(func(err error) (interface{}, error) {
      return nil, fmt.Errorf("there was an error %s", err.Error())
    })

  // do some other stuff

  if value := <-f; value.Error != nil {
    // handle error
  } else {
    result, ok := value.Data.(map[string]interface{})
  }
}
```

With futures.Future.Finally you can specify behavior that should always run regardless of errors

```go
package main

import (
  "github.com/janbialostok/futures"
  "net/http"
  "encoding/json"
  "fmt"
)

func main() {
  f := futures.NewFuture(func() (interface{}, error) {
    return http.Get("website.com")
  }).
    Then(func(value interface{}) (interface{}, error) {
      result, ok := value.Data.(*http.Response)
      if ok {
        var payload map[string]interface{}
        err := json.NewDecoder(result).Decode(&payload)
        return payload, err
      }
      return nil, fmt.Errorf("could not read response")
    }).
    Catch(func(err error) (interface{}, error) {
      return nil, fmt.Errorf("there was an error %s", err.Error())
    }).
    Finally(func() (interface{}, error) {
      return map[string]interface{}{
        "status": "done",
      }, nil
    })

  // do some other stuff

  if value := <-f; value.Error != nil {
    // handle error
  } else {
    result, ok := value.Data.(map[string]interface{})
  }
}
```

futures.Series allows for short hand defintion of a chain of ThenableFuncs

```go
package main

import (
  "github.com/janbialostok/futures"
  "net/http"
  "encoding/json"
  "fmt"
)

func main() {
  f := futures.Series(
    "website.com",
    func(website interface{}) (interface{}, error) {
      return http.Get(website.(string))
    },
    func(value interface{}) (interface{}, error) {
      result, ok := value.Data.(*http.Response)
      if ok {
        var payload map[string]interface{}
        err := json.NewDecoder(result).Decode(&payload)
        return payload, err
      }
      return nil, fmt.Errorf("could not read response")
    },
  )

  // do some other stuff

  if value := <-f; value.Error != nil {
    // handle error
  } else {
    result, ok := value.Data.(map[string]interface{})
  }
}
```

futures.Pipe allows for a Series to be reused with different initial inputs

```go
package main

import (
  "github.com/janbialostok/futures"
  "net/http"
  "encoding/json"
  "fmt"
)

func main() {
  pipe := futures.Pipe(
    func(website interface{}) (interface{}, error) {
      return http.Get(website.(string))
    },
    func(value interface{}) (interface{}, error) {
      result, ok := value.Data.(*http.Response)
      if ok {
        var payload map[string]interface{}
        err := json.NewDecoder(result).Decode(&payload)
        return payload, err
      }
      return nil, fmt.Errorf("could not read response")
    },
  )

  f := pipe("website.com")

  // do some other stuff

  if value := <-f; value.Error != nil {
    // handle error
  } else {
    result, ok := value.Data.(map[string]interface{})
  }
}
```

futures.All allows you to execute and return Value's for any number of Futures, FutureFuns or static values and specify concurrency of execution

```go
package main

import (
  "github.com/janbialostok/futures"
  "net/http"
  "encoding/json"
  "fmt"
)

func main() {
  ops := []interface{}{
    NewFuture(func() (interface{}, error) {
      return http.Get("website.com")
    }),
    func() (interface{}, error) {
      return http.Get("anotherwebsite.com")
    },
  }

  f := futures.All(ops, 2)

  // do some other stuff

  if value := <-f; value.Error != nil {
    // handle error
  } else {
    for _, result := range value.Data.([]interface{}) {

    }
  }
}
```

futures.Map iterates over a slice and executes the specified ThenableFunc with concurrency

```go
package main

import (
  "github.com/janbialostok/futures"
  "net/http"
  "encoding/json"
  "fmt"
)

func main() {
  websites := []interface{}{
    "website.com",
    "anotherwebsite.com",
  }

  f := futures.Map(websites, func(website interface{}) (interface{}, error) {
    return http.Get(website.(string))
  }, 2)

  // do some other stuff

  if value := <-f; value.Error != nil {
    // handle error
  } else {
    for _, result := range value.Data.([]interface{}) {

    }
  }
}
```

## WorkerPool usage

WorkerPool manages concurrency of FutureFunc execution

```go
package main

import (
  "github.com/janbialostok/futures"
  "net/http"
)

func main() {
  pool := futures.NewFuturesWorkerPool(2)
  defer pool.Close()

  for _, website := range []string{"website.com", "anotherwebsite.com"} {
    go func(site string) {
      pool.Send(func() (interface{}, error) {
        return http.Get(site)
      })
    }(website)
  }

  result1, ok := pool.Receive()
  result2, ok := pool.Receive()
}
```

Since order can't be guaranteed for returns futures.WorkerPoolInterface.Do allows for an out channel to be defined if you need access to the return value of a FutureFunc

```go
package main

import (
  "github.com/janbialostok/futures"
  "net/http"
)

func main() {
  pool := futures.NewFuturesWorkerPool(2)
  defer pool.Close()

  result1 := make(chan Value, 1)
  result2 := make(chan Value, 1)
  go func() {
    pool.Do(result1, func() (interface{}, error) {
      return http.Get("website.com")
    })
  }()
  go func() {
    pool.Do(result2, func() (interface{}, error) {
      return http.Get("anotherwebsite.com")
    })
  }()

  result1, ok := <-result1
  result2, ok := <-result2
}
```

This behavior can also be managed using nested pools

```go
package main

import (
  "github.com/janbialostok/futures"
  "net/http"
)

func main() {
  pool := futures.NewFuturesWorkerPool(2)
  subpool1 := pool.Fork(1)
  subpool2 := pool.Fork(1)
  // nested pools can be indepedently closed without closing the parent, but closing the parent will also close any nested pools
  defer pool.Close()

  go func() {
    subpool1.Send(func() (interface{}, error) {
      return http.Get("website.com")
    })
  }()
  go func() {
    subpool2.Send(func() (interface{}, error) {
      return http.Get("anotherwebsite.com")
    })
  }()

  result1, ok := subpool1.Receive()
  result2, ok := subpool2.Receive()
}
```

futures.Map and futures.All can take a WorkerPoolInterface as an argument if you want to manage overall concurrency of your FutureFunc executions

```go
package main

import (
  "github.com/janbialostok/futures"
  "net/http"
)

func main() {
  pool := futures.NewFuturesWorkerPool(2)
  
  f1 := futures.MapWithWorkerPool(
    []interface{}{
      "website.com",
      "anotherwebsite.com",
    },
    func(website interface{}) (interface{}, error) {
      return http.Get(website.(string))
    },
    1,
    pool,
  )

  f2 := futures.AllWithWorkerPool(
    []interface{}{
      func() (interface{}, error) {
        return http.Get("website.com")
      },
      func() (interface{}, error) {
        return http.Get("anotherwebsite.com")
      },
    },
    1,
    pool,
  )

  value, ok := <-f1
  value, ok := <-f2
}
```

