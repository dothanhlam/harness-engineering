// Package fibonacci implements high-performance, arbitrary-precision Fibonacci sequence computations.
// It utilizes an optimized fast doubling algorithm to achieve O(log n) time complexity,
// coupled with thread-safe memoization caching for O(1) retrieval of previously calculated terms.
package fibonacci

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
)

// MaxN represents the upper bound constraint on input n to prevent resource exhaustion (heap/CPU).
const MaxN = 500000

var (
	// ErrNegativeInput is returned when n is less than 0.
	ErrNegativeInput = errors.New("input must be a non-negative integer (n >= 0)")

	// ErrInputTooLarge is returned when n exceeds MaxN to protect against system memory/CPU exhaustion.
	ErrInputTooLarge = fmt.Errorf("input exceeds the maximum supported limit of %d", MaxN)
)

var (
	// cacheMu protects the fibCache map against concurrent access.
	cacheMu sync.RWMutex

	// fibCache serves as a memoization cache storing computed Fibonacci numbers.
	// Initialized with base cases F(0) = 0 and F(1) = 1.
	fibCache = map[int]*big.Int{
		0: big.NewInt(0),
		1: big.NewInt(1),
	}
)

// Fibonacci computes the n-th Fibonacci number.
//
// Mathematical Definition:
//   F(0) = 0, F(1) = 1
//   F(n) = F(n-1) + F(n-2) for n > 1
//
// Complexity:
//   - Time Complexity: O(log n) big integer operations for new calculations, O(1) for cached terms.
//   - Space Complexity: O(log n) stack depth for the calculation, plus O(d) memory where d is the number of digits in F(n).
//
// Resource Management:
//   - The maximum input parameter is capped at MaxN (500,000) to avoid memory or CPU resource exhaustion.
//   - Return values are safely cloned to prevent external mutations from corrupting the memoization cache.
func Fibonacci(n int) (*big.Int, error) {
	if n < 0 {
		return nil, ErrNegativeInput
	}
	if n > MaxN {
		return nil, ErrInputTooLarge
	}

	// 1. Fast path: check cache with a read lock
	cacheMu.RLock()
	if val, ok := fibCache[n]; ok {
		res := new(big.Int).Set(val)
		cacheMu.RUnlock()
		return res, nil
	}
	cacheMu.RUnlock()

	// 2. Slow path: compute values using fast doubling algorithm
	fn, fn1 := pureFibFast(n)

	// 3. Cache the newly computed values under a write lock
	cacheMu.Lock()
	fibCache[n] = new(big.Int).Set(fn)
	fibCache[n+1] = new(big.Int).Set(fn1)
	cacheMu.Unlock()

	// Return a copy to ensure immutability of the cached instance
	return new(big.Int).Set(fn), nil
}

// pureFibFast calculates (F(n), F(n+1)) using the fast doubling method.
// Fast Doubling identities:
//   F(2k) = F(k) * (2*F(k+1) - F(k))
//   F(2k+1) = F(k+1)^2 + F(k)^2
// This function is purely deterministic and side-effect free (no locks or cache writes).
func pureFibFast(n int) (*big.Int, *big.Int) {
	if n == 0 {
		return big.NewInt(0), big.NewInt(1)
	}

	// Recurse on n/2
	a, b := pureFibFast(n / 2)

	// c = F(2k) = a * (2*b - a)
	twoB := new(big.Int).Lsh(b, 1) // 2*b
	twoBMinusA := new(big.Int).Sub(twoB, a)
	c := new(big.Int).Mul(a, twoBMinusA)

	// d = F(2k+1) = b^2 + a^2
	aSq := new(big.Int).Mul(a, a)
	bSq := new(big.Int).Mul(b, b)
	d := new(big.Int).Add(aSq, bSq)

	if n%2 == 0 {
		return c, d
	}
	// F(2k+1) and F(2k+2) where F(2k+2) = F(2k+1) + F(2k)
	return d, new(big.Int).Add(c, d)
}
