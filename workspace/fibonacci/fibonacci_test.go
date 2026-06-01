package fibonacci

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

// TestFibonacciBaseAndStandardCases tests base cases and standard cases of Fibonacci.
func TestFibonacciBaseAndStandardCases(t *testing.T) {
	tests := []struct {
		n        int
		expected *big.Int
	}{
		{0, big.NewInt(0)},
		{1, big.NewInt(1)},
		{2, big.NewInt(1)},
		{3, big.NewInt(2)},
		{4, big.NewInt(3)},
		{5, big.NewInt(5)},
		{10, big.NewInt(55)},
		{20, big.NewInt(6765)},
		{40, big.NewInt(102334155)},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("n_%d", tc.n), func(t *testing.T) {
			res, err := Fibonacci(tc.n)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.expected.Cmp(res) != 0 {
				t.Errorf("expected F(%d) = %s, got %s", tc.n, tc.expected.String(), res.String())
			}
		})
	}
}

// TestFibonacciLargeCases validates large cases, including the transition bounds of uint64 and big integers.
func TestFibonacciLargeCases(t *testing.T) {
	// F(92) is the largest Fibonacci number fitting into a standard uint64.
	f92Expected, ok := new(big.Int).SetString("7540113804746346429", 10)
	if !ok {
		t.Fatal("failed to parse F(92) expected value")
	}

	res92, err := Fibonacci(92)
	if err != nil {
		t.Fatalf("unexpected error for F(92): %v", err)
	}
	if f92Expected.Cmp(res92) != 0 {
		t.Errorf("F(92) mismatch. Expected: %s, got: %s", f92Expected.String(), res92.String())
	}

	// F(100) is a true big.Int value exceeding standard uint64 limits.
	f100Expected, ok := new(big.Int).SetString("354224848179261915075", 10)
	if !ok {
		t.Fatal("failed to parse F(100) expected value")
	}

	res100, err := Fibonacci(100)
	if err != nil {
		t.Fatalf("unexpected error for F(100): %v", err)
	}
	if f100Expected.Cmp(res100) != 0 {
		t.Errorf("F(100) mismatch. Expected: %s, got: %s", f100Expected.String(), res100.String())
	}
}

// TestFibonacciInputValidation verifies input validation guardrails and bounds protection.
func TestFibonacciInputValidation(t *testing.T) {
	t.Run("Negative Inputs", func(t *testing.T) {
		res, err := Fibonacci(-1)
		if err != ErrNegativeInput {
			t.Errorf("expected ErrNegativeInput, got: %v", err)
		}
		if res != nil {
			t.Errorf("expected nil result on error, got: %v", res)
		}

		_, err = Fibonacci(-100)
		if err != ErrNegativeInput {
			t.Errorf("expected ErrNegativeInput, got: %v", err)
		}
	})

	t.Run("Too Large Inputs", func(t *testing.T) {
		res, err := Fibonacci(MaxN + 1)
		if err != ErrInputTooLarge {
			t.Errorf("expected ErrInputTooLarge, got: %v", err)
		}
		if res != nil {
			t.Errorf("expected nil result on error, got: %v", res)
		}
	})
}

// TestFibonacciPropertyBased validates the recurrence relation F(n) = F(n-1) + F(n-2) for random inputs.
func TestFibonacciPropertyBased(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Run 100 randomized property validations
	for i := 0; i < 100; i++ {
		// Keep n reasonably sized to run tests extremely quickly
		n := rng.Intn(1000) + 2 // n in range [2, 1001]

		fn, err := Fibonacci(n)
		if err != nil {
			t.Fatalf("unexpected error for n=%d: %v", n, err)
		}

		fn1, err := Fibonacci(n - 1)
		if err != nil {
			t.Fatalf("unexpected error for n=%d: %v", n-1, err)
		}

		fn2, err := Fibonacci(n - 2)
		if err != nil {
			t.Fatalf("unexpected error for n=%d: %v", n-2, err)
		}

		sum := new(big.Int).Add(fn1, fn2)
		if fn.Cmp(sum) != 0 {
			t.Errorf("Property violation for n=%d: F(n) = %s, F(n-1)+F(n-2) = %s", n, fn.String(), sum.String())
		}
	}
}

// TestFibonacciImmutability ensures returned big.Ints are copies and cannot corrupt the internal cache.
func TestFibonacciImmutability(t *testing.T) {
	val1, err := Fibonacci(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mutate returned big.Int
	val1.SetInt64(99999)

	// Fetch F(5) again
	val2, err := Fibonacci(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val2.Int64() != 5 {
		t.Errorf("Cache corruption detected! F(5) returned %d instead of 5", val2.Int64())
	}
}

// ExampleFibonacci demonstrates standard library integration and usage.
func ExampleFibonacci() {
	result, err := Fibonacci(10)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(result)
	// Output: 55
}

// Benchmarks for performance scalability evaluation.
func BenchmarkFibonacci1000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Reset the cache to measure raw computation performance
		cacheMu.Lock()
		fibCache = map[int]*big.Int{0: big.NewInt(0), 1: big.NewInt(1)}
		cacheMu.Unlock()

		_, _ = Fibonacci(1000)
	}
}

func BenchmarkFibonacci10000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cacheMu.Lock()
		fibCache = map[int]*big.Int{0: big.NewInt(0), 1: big.NewInt(1)}
		cacheMu.Unlock()

		_, _ = Fibonacci(10000)
	}
}

func BenchmarkFibonacci100000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cacheMu.Lock()
		fibCache = map[int]*big.Int{0: big.NewInt(0), 1: big.NewInt(1)}
		cacheMu.Unlock()

		_, _ = Fibonacci(100000)
	}
}
