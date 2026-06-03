// Package factorial provides efficient factorial computation with overflow detection.
package factorial

import (
	"fmt"
	"math"
)

// maxFactorialUint64 is the maximum value of n for which n! fits in uint64
const maxFactorialUint64 = 20

// Factorial computes the factorial of a non-negative integer n (n!).
// It returns n! = n × (n-1) × (n-2) × ... × 1, where 0! = 1.
//
// The function uses an iterative approach for O(n) time complexity and O(1) space complexity.
// It includes overflow detection for uint64 and enforces an upper bound to prevent
// unhandled overflow or excessive computation.
//
// Parameters:
//   - n: A non-negative integer for which to compute the factorial
//
// Returns:
//   - result: The factorial of n as uint64
//   - error: An error if n is negative or if n! would overflow uint64
//
// Examples:
//   factorial.Factorial(0)  // returns 1, nil
//   factorial.Factorial(5)  // returns 120, nil
//   factorial.Factorial(21) // returns 0, error (overflow)
//   factorial.Factorial(-1) // returns 0, error (negative input)
func Factorial(n int) (uint64, error) {
	// Input validation: reject negative integers
	if n < 0 {
		return 0, fmt.Errorf("factorial is not defined for negative integers: %d", n)
	}

	// Enforce upper bound to prevent overflow
	if n > maxFactorialUint64 {
		return 0, fmt.Errorf("factorial(%d) would overflow uint64, maximum supported value is %d", n, maxFactorialUint64)
	}

	// Base cases: 0! = 1 and 1! = 1
	if n <= 1 {
		return 1, nil
	}

	// Iterative computation for O(n) time, O(1) space
	result := uint64(1)
	for i := 2; i <= n; i++ {
		// Additional overflow check during computation
		if result > math.MaxUint64/uint64(i) {
			return 0, fmt.Errorf("factorial(%d) computation would overflow uint64", n)
		}
		result *= uint64(i)
	}

	return result, nil
}

// IsFactorialOverflow checks if computing factorial(n) would overflow uint64.
// This is a utility function for pre-validation without performing the computation.
//
// Parameters:
//   - n: The integer to check
//
// Returns:
//   - bool: true if factorial(n) would overflow uint64, false otherwise
func IsFactorialOverflow(n int) bool {
	return n > maxFactorialUint64
}

// MaxSafeFactorial returns the maximum value of n for which factorial(n)
// can be computed without overflow in uint64.
//
// Returns:
//   - int: The maximum safe value of n (20)
func MaxSafeFactorial() int {
	return maxFactorialUint64
}