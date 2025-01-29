package gopipe

import (
	"fmt"
	"reflect"
	"testing"
)

// Unit Tests
func TestBasicProcess(t *testing.T) {
	proc := func(in <-chan int) (<-chan int, <-chan error) {
		outCh := make(chan int)
		errCh := make(chan error)
		go func() {
			defer close(outCh)
			defer close(errCh)
			for v := range in {
				outCh <- v * 2
			}
		}()
		return outCh, errCh
	}

	p := New(proc)
	input := make(chan int, 3)
	for _, v := range []int{1, 2, 3} {
		input <- v
	}
	close(input)

	results, errors := p.Correct(input)

	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d", len(errors))
	}

	expected := []int{2, 4, 6}
	if !reflect.DeepEqual(results, expected) {
		t.Errorf("expected %v, got %v", expected, results)
	}
}

func TestProcessWithErrors(t *testing.T) {
	proc := func(in <-chan int) (<-chan int, <-chan error) {
		outCh := make(chan int)
		errCh := make(chan error)
		go func() {
			defer close(outCh)
			defer close(errCh)
			for v := range in {
				if v%2 == 0 {
					errCh <- fmt.Errorf("even number %d", v)
				} else {
					outCh <- v * 2
				}
			}
		}()
		return outCh, errCh
	}

	p := New(proc)
	input := make(chan int, 4)
	for _, v := range []int{1, 2, 3, 4} {
		input <- v
	}
	close(input)

	results, errors := p.Correct(input)

	expectedResults := []int{2, 6}
	if !reflect.DeepEqual(results, expectedResults) {
		t.Errorf("expected results %v, got %v", expectedResults, results)
	}

	if len(errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errors))
	}
}

func TestAttachPipelines(t *testing.T) {
	proc1 := func(in <-chan int) (<-chan int, <-chan error) {
		outCh := make(chan int)
		errCh := make(chan error)
		go func() {
			defer close(outCh)
			defer close(errCh)
			for v := range in {
				outCh <- v + 1
			}
		}()
		return outCh, errCh
	}

	proc2 := func(in <-chan int) (<-chan int, <-chan error) {
		outCh := make(chan int)
		errCh := make(chan error)
		go func() {
			defer close(outCh)
			defer close(errCh)
			for v := range in {
				outCh <- v * 2
			}
		}()
		return outCh, errCh
	}

	p1 := New(proc1)
	p2 := Attach(p1, proc2)

	input := make(chan int, 3)
	for _, v := range []int{1, 2, 3} {
		input <- v
	}
	close(input)

	results, errors := p2.Correct(input)

	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d", len(errors))
	}

	expected := []int{4, 6, 8}
	if !reflect.DeepEqual(results, expected) {
		t.Errorf("expected %v, got %v", expected, results)
	}
}
