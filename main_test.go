package gopipe

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

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

	results, errors := p.Collect(input)

	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d", len(errors))
	}

	expected := []int{2, 4, 6}
	if !reflect.DeepEqual(results, expected) {
		t.Errorf("expected %v, got %v", expected, results)
	}
}

func TestConcurrentProcessing(t *testing.T) {
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

	p := New(proc).SetMaxWorkers(3)
	input := make(chan int, 5)
	expected := map[int]struct{}{
		2: {}, 4: {}, 6: {}, 8: {}, 10: {},
	}
	for v := 1; v <= 5; v++ {
		input <- v
	}
	close(input)

	results, errors := p.Collect(input)

	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d", len(errors))
	}

	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}

	for _, res := range results {
		if _, exists := expected[res]; !exists {
			t.Errorf("unexpected result %d", res)
		}
		delete(expected, res)
	}

	if len(expected) > 0 {
		t.Errorf("missing results: %v", expected)
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

	p := New(proc).SetMaxWorkers(2)
	input := make(chan int, 4)
	inputs := []int{1, 2, 3, 4}
	for _, v := range inputs {
		input <- v
	}
	close(input)

	results, errors := p.Collect(input)

	// Verify results (order-independent)
	expectedResults := map[int]struct{}{2: {}, 6: {}}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	for _, res := range results {
		if _, exists := expectedResults[res]; !exists {
			t.Errorf("unexpected result %d", res)
		}
		delete(expectedResults, res)
	}

	// Verify errors (count only due to concurrent nature)
	if len(errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errors))
	}
}

func TestAttachedPipelinesWithConcurrency(t *testing.T) {
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

	p1 := New(proc1).SetMaxWorkers(2)
	p2 := Attach(p1, proc2).SetMaxWorkers(2)

	input := make(chan int, 5)
	expected := map[int]struct{}{
		4:  {}, // (1+1)*2
		6:  {}, // (2+1)*2
		8:  {}, // (3+1)*2
		10: {}, // (4+1)*2
		12: {}, // (5+1)*2
	}
	for v := 1; v <= 5; v++ {
		input <- v
	}
	close(input)

	results, errors := p2.Collect(input)

	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d", len(errors))
	}

	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}

	for _, res := range results {
		if _, exists := expected[res]; !exists {
			t.Errorf("unexpected result %d", res)
		}
		delete(expected, res)
	}

	if len(expected) > 0 {
		t.Errorf("missing results: %v", expected)
	}
}

func TestOrderPreservation(t *testing.T) {
	proc := func(in <-chan int) (<-chan int, <-chan error) {
		outCh := make(chan int)
		errCh := make(chan error)
		go func() {
			defer close(outCh)
			defer close(errCh)
			for v := range in {
				outCh <- v
			}
		}()
		return outCh, errCh
	}

	p := New(proc).SetMaxWorkers(1)
	input := make(chan int, 5)
	numbers := []int{5, 3, 1, 4, 2}
	for _, v := range numbers {
		input <- v
	}
	close(input)

	results, _ := p.Collect(input)

	// Should preserve order with single worker
	if !reflect.DeepEqual(results, numbers) {
		t.Errorf("expected %v, got %v", numbers, results)
	}
}

func TestBatchProcess(t *testing.T) {
	t.Run("split batch", func(t *testing.T) {
		input := make(chan int)
		batchSize := 3

		go func() {
			defer close(input)
			for i := 1; i <= 7; i++ {
				input <- i
			}
		}()

		outCh, _ := BatchProcess[int](batchSize)(input)
		results := collectBatches(outCh)

		expected := [][]int{
			{1, 2, 3},
			{4, 5, 6},
			{7},
		}

		if !reflect.DeepEqual(results, expected) {
			t.Errorf("Expected %v, got %v", expected, results)
		}
	})

	t.Run("blank", func(t *testing.T) {
		input := make(chan int)
		close(input)

		outCh, _ := BatchProcess[int](2)(input)
		results := collectBatches(outCh)

		if len(results) != 0 {
			t.Errorf("Expected empty result, got %v", results)
		}
	})

	t.Run("batch size", func(t *testing.T) {
		input := make(chan int)
		go func() {
			defer close(input)
			input <- 1
			input <- 2
		}()

		outCh, _ := BatchProcess[int](1)(input)
		results := collectBatches(outCh)

		expected := [][]int{{1}, {2}}
		if !reflect.DeepEqual(results, expected) {
			t.Errorf("Expected %v, got %v", expected, results)
		}
	})

	t.Run("error cascade", func(t *testing.T) {
		errProc := func(in <-chan int) (<-chan int, <-chan error) {
			out := make(chan int)
			errCh := make(chan error, 1)
			go func() {
				defer close(out)
				defer close(errCh)
				errCh <- errors.New("test error")
			}()
			return out, errCh
		}

		pipeline := WithBatch(New(errProc), 2)
		_, errCh := pipeline.Run(make(chan int))

		select {
		case err := <-errCh:
			if err.Error() != "test error" {
				t.Errorf("Expected 'test error', got %v", err)
			}
		case <-time.After(1 * time.Second):
			t.Error("Error not received")
		}
	})
}

func collectBatches[T any](ch <-chan []T) [][]T {
	var batches [][]T
	for batch := range ch {
		batches = append(batches, batch)
	}
	return batches
}
