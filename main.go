package gopipe

import (
	"sync"
)

type Process[In, Out any] func(<-chan In) (<-chan Out, <-chan error)

type Pipeline[In, Out any] struct {
	proc Process[In, Out]
}

func New[In, Out any](proc Process[In, Out]) *Pipeline[In, Out] {
	return &Pipeline[In, Out]{
		proc: proc,
	}
}

func Attach[In, Mid, Out any](
	prev *Pipeline[In, Mid],
	nextProc Process[Mid, Out],
) *Pipeline[In, Out] {
	return &Pipeline[In, Out]{
		proc: chainProcess(prev.proc, nextProc),
	}
}

func chainProcess[In, Mid, Out any](
	prevProc Process[In, Mid],
	nextProc Process[Mid, Out],
) Process[In, Out] {
	return func(in <-chan In) (<-chan Out, <-chan error) {
		midCh, errCh1 := prevProc(in)
		outCh, errCh2 := nextProc(midCh)
		return outCh, mergeErrors(errCh1, errCh2)
	}
}

func (p *Pipeline[In, Out]) Run(input <-chan In) (<-chan Out, <-chan error) {
	if p.proc == nil {
		panic("no process defined")
	}
	return p.proc(input)
}

func (p *Pipeline[In, Out]) Correct(input <-chan In) ([]Out, []error) {
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

func mergeErrors(chs ...<-chan error) <-chan error {
	out := make(chan error)
	var wg sync.WaitGroup

	for _, ch := range chs {
		if ch == nil {
			continue
		}
		wg.Add(1)
		go func(c <-chan error) {
			defer wg.Done()
			for err := range c {
				out <- err
			}
		}(ch)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
