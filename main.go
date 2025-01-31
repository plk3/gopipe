package gopipe

import (
	"sync"
)

type Process[In, Out any] func(<-chan In) (<-chan Out, <-chan error)

type Pipeline[In, Out any] struct {
	proc       Process[In, Out]
	maxWorkers int
}

func New[In, Out any](proc Process[In, Out]) *Pipeline[In, Out] {
	return &Pipeline[In, Out]{
		proc:       proc,
		maxWorkers: 1,
	}
}

func Attach[In, Mid, Out any](
	prev *Pipeline[In, Mid],
	nextProc Process[Mid, Out],
) *Pipeline[In, Out] {
	return &Pipeline[In, Out]{
		proc:       chainProcess(prev.proc, nextProc),
		maxWorkers: prev.maxWorkers,
	}
}

func chainProcess[In, Mid, Out any](
	prevProc Process[In, Mid],
	nextProc Process[Mid, Out],
) Process[In, Out] {
	return func(in <-chan In) (<-chan Out, <-chan error) {
		midCh, errCh1 := prevProc(in)
		outCh, errCh2 := nextProc(midCh)
		return outCh, fanIn([]<-chan error{errCh1, errCh2})
	}
}

func (p *Pipeline[In, Out]) SetMaxWorkers(n int) *Pipeline[In, Out] {
	if n < 1 {
		panic("max workers must be at least 1")
	}
	p.maxWorkers = n
	return p
}

func (p *Pipeline[In, Out]) Run(input <-chan In) (<-chan Out, <-chan error) {
	if p.proc == nil {
		panic("no process defined")
	}

	if p.maxWorkers <= 1 {
		return p.proc(input)
	}

	fanOutChs := fanOut(input, p.maxWorkers)

	var outChs []<-chan Out
	var errChs []<-chan error
	for _, ch := range fanOutChs {
		out, err := p.proc(ch)
		outChs = append(outChs, out)
		errChs = append(errChs, err)
	}

	mergedOut := fanIn(outChs)
	mergedErr := fanIn(errChs)

	return mergedOut, mergedErr
}

func (p *Pipeline[In, Out]) Collect(input <-chan In) ([]Out, []error) {
	outputCh, errorCh := p.Run(input)

	var results []Out
	var errors []error
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for out := range outputCh {
			results = append(results, out)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for e := range errorCh {
			errors = append(errors, e)
		}
	}()

	wg.Wait()

	return results, errors
}

func fanOut[In any](in <-chan In, n int) []<-chan In {
	chs := make([]chan In, n)
	for i := range chs {
		chs[i] = make(chan In)
	}

	go func() {
		defer func() {
			for _, ch := range chs {
				close(ch)
			}
		}()

		i := 0
		for v := range in {
			chs[i] <- v
			i = (i + 1) % n
		}
	}()

	outChs := make([]<-chan In, n)
	for i, ch := range chs {
		outChs[i] = ch
	}
	return outChs
}

func fanIn[Out any](chs []<-chan Out) <-chan Out {
	out := make(chan Out)
	var wg sync.WaitGroup

	for _, ch := range chs {
		wg.Add(1)
		go func(c <-chan Out) {
			defer wg.Done()
			for v := range c {
				out <- v
			}
		}(ch)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func BatchProcess[In any](batchSize int) Process[In, []In] {
	return func(in <-chan In) (<-chan []In, <-chan error) {
		out := make(chan []In)
		errCh := make(chan error, 1)

		go func() {
			defer close(out)
			defer close(errCh)

			var batch []In
			for item := range in {
				batch = append(batch, item)
				if len(batch) >= batchSize {
					out <- batch
					batch = nil
				}
			}

			if len(batch) > 0 {
				out <- batch
			}
		}()

		return out, errCh
	}
}

func WithBatch[In, Out any](
	p *Pipeline[In, Out],
	batchSize int,
) *Pipeline[In, []Out] {
	batchProc := func(in <-chan Out) (<-chan []Out, <-chan error) {
		return BatchProcess[Out](batchSize)(in)
	}
	return Attach(p, batchProc)
}
