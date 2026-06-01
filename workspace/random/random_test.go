package random

import (
	"bytes"
	"math"
	"sync"
	"testing"
)

// TestIntRangeSuccess checks correct generation of integers within bounds, including large integers and negative bounds.
func TestIntRangeSuccess(t *testing.T) {
	tests := []struct {
		name string
		min  int
		max  int
	}{
		{"positive range", 10, 20},
		{"negative range", -20, -10},
		{"mixed range", -10, 10},
		{"single value range", 5, 5},
		{"large range", -1000000, 1000000},
		{"extreme full range", math.MinInt, math.MaxInt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 100; i++ {
				val, err := IntRange(tt.min, tt.max)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if val < tt.min || val > tt.max {
					t.Errorf("generated value %d out of bounds [%d, %d]", val, tt.min, tt.max)
				}
			}
		})
	}
}

// TestIntRangeInvalidBounds verifies error handling when min > max.
func TestIntRangeInvalidBounds(t *testing.T) {
	_, err := IntRange(10, 5)
	if err == nil {
		t.Error("expected error when min > max, got nil")
	}
	if err != ErrMinGreaterThanMax {
		t.Errorf("expected ErrMinGreaterThanMax, got %v", err)
	}
}

// TestFloat checks that floating point generation satisfies [0.0, 1.0).
func TestFloat(t *testing.T) {
	for i := 0; i < 1000; i++ {
		val := Float()
		if val < 0.0 || val >= 1.0 {
			t.Errorf("generated float %f out of bounds [0.0, 1.0)", val)
		}
	}
}

// TestBytes checks generation of random byte slices, including empty and negative size requests.
func TestBytes(t *testing.T) {
	t.Run("valid size", func(t *testing.T) {
		size := 32
		b, err := Bytes(size)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(b) != size {
			t.Errorf("expected byte slice of length %d, got %d", size, len(b))
		}
	})

	t.Run("zero size", func(t *testing.T) {
		b, err := Bytes(0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(b) != 0 {
			t.Errorf("expected empty slice, got length %d", len(b))
		}
	})

	t.Run("negative size", func(t *testing.T) {
		_, err := Bytes(-5)
		if err == nil {
			t.Error("expected error for negative length, got nil")
		}
		if err != ErrNegativeLength {
			t.Errorf("expected ErrNegativeLength, got %v", err)
		}
	})
}

// TestShuffleBytes checks in-place shuffling of a byte slice.
func TestShuffleBytes(t *testing.T) {
	t.Run("normal slice", func(t *testing.T) {
		original := []byte("abcdefghijklmnopqrstuvwxyz")
		shuffled := make([]byte, len(original))
		copy(shuffled, original)

		ShuffleBytes(shuffled)

		if len(shuffled) != len(original) {
			t.Fatalf("length changed after shuffle")
		}

		// Verify all original elements are present.
		counts := make(map[byte]int)
		for _, b := range original {
			counts[b]++
		}
		for _, b := range shuffled {
			counts[b]--
		}
		for b, count := range counts {
			if count != 0 {
				t.Errorf("mismatch in element counts for byte %c", b)
			}
		}
	})

	t.Run("empty and small slices", func(t *testing.T) {
		ShuffleBytes(nil)
		ShuffleBytes([]byte{})
		
		single := []byte{42}
		ShuffleBytes(single)
		if single[0] != 42 {
			t.Error("1-byte slice modified after shuffle")
		}
	})
}

// TestConcurrency runs multiple goroutines calling PRNG functions to verify thread safety.
func TestConcurrency(t *testing.T) {
	var wg sync.WaitGroup
	workers := 50
	iterations := 200

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, _ = IntRange(0, 100)
				_ = Float()
				_, _ = Bytes(16)
				
				b := []byte("hello concurrent world")
				ShuffleBytes(b)
			}
		}()
	}
	wg.Wait()
}

// TestCryptoBytesSuccess checks success behavior of CSPRNG bytes.
func TestCryptoBytesSuccess(t *testing.T) {
	t.Run("valid size", func(t *testing.T) {
		b, err := CryptoBytes(32)
		if err != nil {
			t.Fatalf("unexpected CSPRNG error: %v", err)
		}
		if len(b) != 32 {
			t.Errorf("expected 32 bytes, got %d", len(b))
		}

		// Two consecutive calls should be highly unlikely to return the same byte slice.
		b2, _ := CryptoBytes(32)
		if bytes.Equal(b, b2) {
			t.Error("CSPRNG returned duplicate byte slices")
		}
	})

	t.Run("zero size", func(t *testing.T) {
		b, err := CryptoBytes(0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(b) != 0 {
			t.Errorf("expected empty slice, got length %d", len(b))
		}
	})

	t.Run("negative size", func(t *testing.T) {
		_, err := CryptoBytes(-1)
		if err == nil {
			t.Error("expected error for negative length, got nil")
		}
		if err != ErrNegativeLength {
			t.Errorf("expected ErrNegativeLength, got %v", err)
		}
	})
}

// TestCryptoIntRangeSuccess checks CSPRNG integer generation limits.
func TestCryptoIntRangeSuccess(t *testing.T) {
	t.Run("valid range", func(t *testing.T) {
		min, max := 1000, 2000
		for i := 0; i < 100; i++ {
			val, err := CryptoIntRange(min, max)
			if err != nil {
				t.Fatalf("unexpected CSPRNG error: %v", err)
			}
			if val < min || val > max {
				t.Errorf("CSPRNG value %d out of bounds [%d, %d]", val, min, max)
			}
		}
	})

	t.Run("single value range", func(t *testing.T) {
		val, err := CryptoIntRange(42, 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != 42 {
			t.Errorf("expected 42, got %d", val)
		}
	})

	t.Run("invalid bounds", func(t *testing.T) {
		_, err := CryptoIntRange(10, 5)
		if err == nil {
			t.Error("expected error when min > max, got nil")
		}
		if err != ErrMinGreaterThanMax {
			t.Errorf("expected ErrMinGreaterThanMax, got %v", err)
		}
	})
}

// TestCryptoFloat checks that CSPRNG float generation is within [0.0, 1.0).
func TestCryptoFloat(t *testing.T) {
	for i := 0; i < 1000; i++ {
		val, err := CryptoFloat()
		if err != nil {
			t.Fatalf("unexpected CSPRNG error: %v", err)
		}
		if val < 0.0 || val >= 1.0 {
			t.Errorf("CSPRNG float %f out of bounds [0.0, 1.0)", val)
		}
	}
}

// TestChiSquaredUniformDistribution uses Chi-squared goodness-of-fit to validate uniform distribution.
func TestChiSquaredUniformDistribution(t *testing.T) {
	k := 10       // number of bins (0 to 9)
	n := 10000    // total samples
	expected := float64(n) / float64(k)

	observed := make([]int, k)
	for i := 0; i < n; i++ {
		val, err := IntRange(0, k-1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		observed[val]++
	}

	// Chi-squared calculation
	var chi2 float64
	for _, obs := range observed {
		diff := float64(obs) - expected
		chi2 += (diff * diff) / expected
	}

	// Degrees of freedom = k - 1 = 9
	// For df = 9, critical value at significance level alpha = 0.01 is 21.67
	criticalValue := 21.67
	if chi2 > criticalValue {
		t.Errorf("Chi-squared test failed: chi2 = %f (expected < %f). Distribution might not be uniform.", chi2, criticalValue)
	}
}

// TestChiSquaredCryptoUniformDistribution uses Chi-squared goodness-of-fit to validate CSPRNG uniform distribution.
func TestChiSquaredCryptoUniformDistribution(t *testing.T) {
	k := 10
	n := 10000
	expected := float64(n) / float64(k)

	observed := make([]int, k)
	for i := 0; i < n; i++ {
		val, err := CryptoIntRange(0, k-1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		observed[val]++
	}

	var chi2 float64
	for _, obs := range observed {
		diff := float64(obs) - expected
		chi2 += (diff * diff) / expected
	}

	criticalValue := 21.67
	if chi2 > criticalValue {
		t.Errorf("Chi-squared CSPRNG test failed: chi2 = %f (expected < %f). Distribution might not be uniform.", chi2, criticalValue)
	}
}

// TestCustomSeededLocalInstance verifies that custom seeded instances generate predictable sequences.
func TestCustomSeededLocalInstance(t *testing.T) {
	// Creating two independent generators with the exact same non-zero seed should produce identical outputs.
	g1 := NewGenerator(12345, 67890)
	g2 := NewGenerator(12345, 67890)

	for i := 0; i < 50; i++ {
		v1, err1 := g1.IntRange(0, 100000)
		v2, err2 := g2.IntRange(0, 100000)
		if err1 != nil || err2 != nil {
			t.Fatalf("unexpected error: %v, %v", err1, err2)
		}
		if v1 != v2 {
			t.Fatalf("predictability test failed at iteration %d: %d != %d", i, v1, v2)
		}
	}

	// Creating a generator with a different seed should produce a different sequence.
	g3 := NewGenerator(54321, 98760)
	different := false
	for i := 0; i < 50; i++ {
		v1, _ := g1.IntRange(0, 100000)
		v3, _ := g3.IntRange(0, 100000)
		if v1 != v3 {
			different = true
			break
		}
	}
	if !different {
		t.Error("expected different seeds to produce different sequences")
	}
}

// TestAutoSeeding verifies that NewGenerator(0, 0) triggers auto-seeding correctly.
func TestAutoSeeding(t *testing.T) {
	// Using 0, 0 seeds automatically. Two calls should yield different generators (highly improbable to have same seeds)
	g1 := NewGenerator(0, 0)
	g2 := NewGenerator(0, 0)

	different := false
	for i := 0; i < 100; i++ {
		v1, _ := g1.IntRange(0, 1000000)
		v2, _ := g2.IntRange(0, 1000000)
		if v1 != v2 {
			different = true
			break
		}
	}
	if !different {
		t.Error("expected auto-seeded generators to yield different sequences")
	}
}

// ─────────────────────────────────────────────
// BENCHMARKS
// ─────────────────────────────────────────────

func BenchmarkIntRange(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = IntRange(0, 1000)
	}
}

func BenchmarkFloat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Float()
	}
}

func BenchmarkBytes32(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Bytes(32)
	}
}
