# gopipe
gopipe is a Go library for building concurrent data processing pipelines with automatic error aggregation. It allows you to chain processing stages together while handling concurrency and error propagation seamlessly.

## Features
- Concurrent pipeline processing
- Automatic error aggregation from all stages
- Type-safe generics for input/output types
- Simple API for building complex pipelines
- Composable pipeline stages

## Installation
```bash
go get github.com/plk3/gopipe
```

## Sample
```go
package main

import (
	"fmt"
    "runtime"
	"github.com/plk3/gopipe"
)

func main() {
    // Create a pipeline that multiplies numbers and then squares them
    multiplier := gopipe.New(multiplierProcess)
    pipeline := gopipe.Attach(multiplier, squarerProcess).SetMaxWorkers(runtime.NumCPU())

    // Create input channel
    inputCh := make(chan int)
    go func() {
        defer close(inputCh)
        for i := 1; i <= 5; i++ {
            inputCh <- i
        }
    }()

    // Start the pipeline
    output, errors := pipeline.Collect(inputCh)

    fmt.Println(output)
    fmt.Println(errors)
}

func multiplierProcess(in <-chan int) (<-chan int, <-chan error) {
    out := make(chan int)
    errCh := make(chan error)
    go func() {
        defer close(out)
        defer close(errCh)
        for num := range in {
            out <- num * 2
        }
    }()
    return out, errCh
}

func squarerProcess(in <-chan int) (<-chan int, <-chan error) {
    out := make(chan int)
    errCh := make(chan error)
    go func() {
        defer close(out)
        defer close(errCh)
        for num := range in {
            out <- num * num
        }
    }()
    return out, errCh
}
```

## API Documentation
### Process[In, Out any]
```go
type Process[In, Out any] func(<-chan In) (<-chan Out, <-chan error)
A processing stage that takes an input channel and returns output and error channels.
```

### Pipeline[In, Out any]
```go
type Pipeline[In, Out any] struct {
    // contains filtered or unexported fields
}
```

## Functions
### New[In, Out any]
```go
func New[In, Out any](proc Process[In, Out]) *Pipeline[In, Out]
// Creates a new pipeline with the initial processing stage.
```

### Attach[In, Mid, Out any]
```go
func Attach[In, Mid, Out any](
    prev *Pipeline[In, Mid],
    nextProc Process[Mid, Out],
) *Pipeline[In, Out]
// Attaches a new processing stage to an existing pipeline.
```

## Methods
### (p *Pipeline[In, Out]) Run
```go
func (p *Pipeline[In, Out]) Run(input <-chan In) (<-Out, <-error)
// Starts the pipeline with the given input channel, returning output and error channel.
```
### (p *Pipeline[In, Out]) Collect
```go
func (p *Pipeline[In, Out]) Collect(input <-chan In) ([]Out, []error)
// Starts the pipeline with the given input channel, returning output and error slice.
```