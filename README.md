# gopipe
gopipe is a Go library for building concurrent data processing pipelines with automatic error aggregation. It enables easy composition of pipeline stages while handling concurrency and error propagation.

## Features
- **Concurrent pipeline processing**
- Configurable maximum workers per stage
- Non-deterministic output order (when using multiple workers)
- Automatic error aggregation from all stages
- Type-safe generics for input/output types
- Simple API for complex pipeline construction
- Composable pipeline stages
- Safe channel closure management

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

    fmt.Println(output) // Example: [4 16 36 64 100] (order may vary)
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