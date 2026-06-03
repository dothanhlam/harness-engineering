package factorial

import (
	"testing"
)

// TestFactorialBaseCases tests the fundamental base cases: 0! and 1!
func TestFactorialBaseCases(t *testing.T) {
	tests := []struct {
		input    int
		expected uint64
		name     string
	}{
		{0, 1, "factorial of 0"},
		{1, 1, "factorial of 1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Factorial(test.input)
			if err != nil {
				t.Errorf("Factorial(%d) returned unexpected error: %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("Factorial(%d) = %d, expected %d", test.input, result, test.expected)
			}
		})
	}
}

// TestFactorialStandardCases tests standard factorial computations
func TestFactorialStandardCases(t *testing.T) {
	tests := []struct {
		input    int
		expected uint64
		name     string
	}{
		{2, 2, "factorial of 2"},
		{3, 6, "factorial of 3"},
		{4, 24, "factorial of 4"},
		{5, 120, "factorial of 5"},
		{6, 720, "factorial of 6"},
		{10, 3628800, "factorial of 10"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Factorial(test.input)
			if err != nil {
				t.Errorf("Factorial(%d) returned unexpected error: %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("Factorial(%d) = %d, expected %d", test.input, result, test.expected)
			}
		})
	}
}

// TestFactorialBoundaryCases tests the maximum value before overflow
func TestFactorialBoundaryCases(t *testing.T) {
	// Test the maximum safe value (20! = 2432902008176640000)
	result, err := Factorial(20)
	if err != nil {
		t.Errorf("Factorial(20) returned unexpected error: %v", err)
	}
	expected := uint64(2432902008176640000)
	if result != expected {
		t.Errorf("Factorial(20) = %d, expected %d", result, expected)
	}

	// Test that 21 triggers overflow error
	_, err = Factorial(21)
	if err == nil {
		t.Error("Factorial(21) should return an overflow error")
	}
}

// TestFactorialNegativeInput tests error handling for negative inputs
func TestFactorialNegativeInput(t *testing.T) {
	negativeInputs := []int{-1, -5, -100}

	for _, input := range negativeInputs {
		t.Run("negative_input", func(t *testing.T) {
			result, err := Factorial(input)
			if err == nil {
				t.Errorf("Factorial(%d) should return an error for negative input", input)
			}
			if result != 0 {
				t.Errorf("Factorial(%d) should return 0 for error cases, got %d", input, result)
			}
		})
	}
}

// TestFactorialOverflowCases tests overflow detection for large inputs
func TestFactorialOverflowCases(t *testing.T) {
	overflowInputs := []int{21, 25, 30, 100}

	for _, input := range overflowInputs {
		t.Run("overflow_input", func(t *testing.T) {
			result, err := Factorial(input)
			if err == nil {
				t.Errorf("Factorial(%d) should return an overflow error", input)
			}
			if result != 0 {
				t.Errorf("Factorial(%d) should return 0 for error cases, got %d", input, result)
			}
		})
	}
}

// TestIsFactorialOverflow tests the overflow checking utility function
func TestIsFactorialOverflow(t *testing.T) {
	tests := []struct {
		input    int
		expected bool
		name     string
	}{
		{0, false, "0 does not overflow"},
		{10, false, "10 does not overflow"},
		{20, false, "20 does not overflow"},
		{21, true, "21 would overflow"},
		{100, true, "100 would overflow"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsFactorialOverflow(test.input)
			if result != test.expected {
				t.Errorf("IsFactorialOverflow(%d) = %t, expected %t", test.input, result, test.expected)
			}
		})
	}
}

// TestMaxSafeFactorial tests the utility function that returns the maximum safe value
func TestMaxSafeFactorial(t *testing.T) {
	expected := 20
	result := MaxSafeFactorial()
	if result != expected {
		t.Errorf("MaxSafeFactorial() = %d, expected %d", result, expected)
	}
}

// TestFactorialSequentialConsistency verifies mathematical correctness through sequential computation
func TestFactorialSequentialConsistency(t *testing.T) {
	// Verify that factorial(n+1) = factorial(n) * (n+1)
	for n := 0; n < 19; n++ { // Test up to 19 to avoid overflow on n+1
		factN, err1 := Factorial(n)
		factNPlus1, err2 := Factorial(n + 1)

		if err1 != nil || err2 != nil {
			t.Errorf("Unexpected errors: Factorial(%d)=%v, Factorial(%d)=%v", n, err1, n+1, err2)
			continue
		}

		expected := factN * uint64(n+1)
		if factNPlus1 != expected {
			t.Errorf("Sequential consistency failed: Factorial(%d+1) = %d, expected %d * %d = %d",
				n, factNPlus1, factN, n+1, expected)
		}
	}
}

// BenchmarkFactorial benchmarks factorial computation for various input sizes
func BenchmarkFactorial(b *testing.B) {
	inputs := []int{5, 10, 15, 20}

	for _, input := range inputs {
		b.Run("n="+string(rune(input+'0')), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = Factorial(input)
			}
		})
	}
}

// BenchmarkFactorialSmall benchmarks small factorial computations
func BenchmarkFactorialSmall(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Factorial(5)
	}
}

// BenchmarkFactorialLarge benchmarks the largest safe factorial computation
func BenchmarkFactorialLarge(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Factorial(20)
	}
}