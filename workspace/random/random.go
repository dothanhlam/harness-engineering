// Package random implements thread-safe pseudo-random (PRNG) and cryptographically secure (CSPRNG)
// random number generation, compliant with project-specific security rules and modern Go standards.
package random

import (
	crypto_rand "crypto/rand"
	"errors"
	"math/big"
	"math/rand/v2"
	"sync"
	"time"
)

var (
	// ErrMinGreaterThanMax is returned when the min boundary is strictly greater than the max boundary.
	ErrMinGreaterThanMax = errors.New("min boundary cannot be greater than max boundary")

	// ErrNegativeLength is returned when a request is made for a negative number of random bytes.
	ErrNegativeLength = errors.New("length cannot be negative")
)

// Generator wraps a pseudo-random number generator (PRNG) source and provides
// thread-safe operations on top of it using mutual exclusion.
type Generator struct {
	mu   sync.Mutex
	prng *rand.Rand
}

var (
	globalGen *Generator
	once      sync.Once
)

// NewGenerator creates a new custom-seeded Generator.
// If both seed1 and seed2 are 0, it seeds automatically using high-entropy sources (crypto/rand with fallback to time.Now().UnixNano()).
func NewGenerator(seed1, seed2 uint64) *Generator {
	if seed1 == 0 && seed2 == 0 {
		var b [16]byte
		_, err := crypto_rand.Read(b[:])
		if err != nil {
			// Fallback to time-based high-entropy seed if CSPRNG is unavailable.
			t := uint64(time.Now().UnixNano())
			seed1 = t
			seed2 = t ^ 0x5555555555555555
		} else {
			seed1 = uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
				uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
			seed2 = uint64(b[8])<<56 | uint64(b[9])<<48 | uint64(b[10])<<40 | uint64(b[11])<<32 |
				uint64(b[12])<<24 | uint64(b[13])<<16 | uint64(b[14])<<8 | uint64(b[15])
		}
	}
	src := rand.NewPCG(seed1, seed2)
	return &Generator{
		prng: rand.New(src),
	}
}

// Global returns the global singleton Generator instance, initialized lazily.
func Global() *Generator {
	once.Do(func() {
		globalGen = NewGenerator(0, 0)
	})
	return globalGen
}

// IntRange generates a pseudo-random integer in the range [min, max] inclusive.
// It returns an error if min > max.
func (g *Generator) IntRange(min, max int) (int, error) {
	if min > max {
		return 0, ErrMinGreaterThanMax
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	diff := uint64(max) - uint64(min)
	if diff == 0 {
		return min, nil
	}

	var val uint64
	if diff == ^uint64(0) {
		val = g.prng.Uint64()
	} else {
		val = g.prng.Uint64N(diff + 1)
	}

	return int(uint64(min) + val), nil
}

// Float generates a pseudo-random float64 in the range [0.0, 1.0).
func (g *Generator) Float() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.prng.Float64()
}

// Bytes generates a slice of n pseudo-random bytes.
// It returns an error if n is negative.
func (g *Generator) Bytes(n int) ([]byte, error) {
	if n < 0 {
		return nil, ErrNegativeLength
	}
	if n == 0 {
		return []byte{}, nil
	}
	b := make([]byte, n)
	g.mu.Lock()
	defer g.mu.Unlock()

	// Fill the slice in chunks of 8 bytes (uint64)
	for i := 0; i < n; {
		val := g.prng.Uint64()
		for j := 0; j < 8 && i < n; j++ {
			b[i] = byte(val)
			val >>= 8
			i++
		}
	}
	return b, nil
}

// ShuffleBytes shuffles a slice of bytes in-place using pseudo-random generation.
func (g *Generator) ShuffleBytes(b []byte) {
	if len(b) <= 1 {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.prng.Shuffle(len(b), func(i, j int) {
		b[i], b[j] = b[j], b[i]
	})
}

// ─────────────────────────────────────────────
// CONVENIENCE GLOBAL FUNCTIONS
// ─────────────────────────────────────────────

// IntRange generates a pseudo-random integer in the range [min, max] inclusive using the global generator.
func IntRange(min, max int) (int, error) {
	return Global().IntRange(min, max)
}

// Float generates a pseudo-random float64 in the range [0.0, 1.0) using the global generator.
func Float() float64 {
	return Global().Float()
}

// Bytes generates a slice of n pseudo-random bytes using the global generator.
func Bytes(n int) ([]byte, error) {
	return Global().Bytes(n)
}

// ShuffleBytes shuffles a slice of bytes in-place using the global generator.
func ShuffleBytes(b []byte) {
	Global().ShuffleBytes(b)
}

// ─────────────────────────────────────────────
// CRYPTOGRAPHICALLY SECURE FUNCTIONS (CSPRNG)
// ─────────────────────────────────────────────

// CryptoBytes generates cryptographically secure random bytes of specified length using crypto/rand.
// It returns an error if the length is negative or if the system's entropy source fails.
func CryptoBytes(n int) ([]byte, error) {
	if n < 0 {
		return nil, ErrNegativeLength
	}
	if n == 0 {
		return []byte{}, nil
	}
	b := make([]byte, n)
	_, err := crypto_rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// CryptoIntRange generates a cryptographically secure random integer in the range [min, max] inclusive using crypto/rand.
// It returns an error if min > max or if the system's entropy source fails.
func CryptoIntRange(min, max int) (int, error) {
	if min > max {
		return 0, ErrMinGreaterThanMax
	}
	diff := uint64(max) - uint64(min)
	if diff == 0 {
		return min, nil
	}

	limit := new(big.Int).SetUint64(diff + 1)
	n, err := crypto_rand.Int(crypto_rand.Reader, limit)
	if err != nil {
		return 0, err
	}

	return int(uint64(min) + n.Uint64()), nil
}

// CryptoFloat generates a cryptographically secure random float64 in the range [0.0, 1.0) using crypto/rand.
// It returns an error if the system's entropy source fails.
func CryptoFloat() (float64, error) {
	var b [8]byte
	_, err := crypto_rand.Read(b[:])
	if err != nil {
		return 0.0, err
	}

	// 53 bits of precision for float64.
	val := uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])

	val = val & 0x1FFFFFFFFFFFFF // mask to 53 bits
	f := float64(val) / 9007199254740992.0

	return f, nil
}
