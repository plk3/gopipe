package gopipe

import (
	"math"
	"runtime"
	"testing"
)

var numElements = 100

func heavyTask(x int) float64 {
	result := math.Sqrt(math.Pow(float64(x), 3.5)) * math.Sin(float64(x))
	for i := 0; i < 1000000; i++ {
		result += math.Pow(math.Sin(float64(i)), 2) * math.Cos(float64(i))
	}

	return result
}

func BenchmarkSequentialHeavyTask(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := make([]float64, numElements)
		for j := 0; j < numElements; j++ {
			results[j] = heavyTask(j)
		}
	}
}

func BenchmarkParallelHeavyTask(b *testing.B) {
	proc := func(in <-chan int) (<-chan float64, <-chan error) {
		outCh := make(chan float64)
		errCh := make(chan error)
		go func() {
			defer close(outCh)
			defer close(errCh)
			for v := range in {
				outCh <- heavyTask(v)
			}
		}()
		return outCh, errCh
	}

	p := New(proc).SetMaxWorkers(runtime.NumCPU())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := make(chan int)
		go func() {
			defer close(input)
			for j := 0; j < numElements; j++ {
				input <- j
			}
		}()
		p.Correct(input)
	}
}
