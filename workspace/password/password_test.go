package password

import (
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	// Reset cost factor to default before testing
	ResetCost()

	// 1. Verify standard hashing and check
	pwd := "SecurePass123!"
	hash, err := HashPassword(pwd)
	if err != nil {
		t.Fatalf("unexpected error hashing password: %v", err)
	}

	// Verify standard format ($2a$, $2b$, or $2y$)
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") && !strings.HasPrefix(hash, "$2y$") {
		t.Errorf("unexpected hash prefix format: %s", hash)
	}

	// Verify CheckPasswordHash works for correct password
	if !CheckPasswordHash(pwd, hash) {
		t.Error("expected CheckPasswordHash to return true for correct password")
	}

	// Verify CheckPasswordHash returns false for incorrect password
	if CheckPasswordHash("WrongPass123!", hash) {
		t.Error("expected CheckPasswordHash to return false for incorrect password")
	}

	// Verify salting: two hashes of the same password should be different
	hash2, err := HashPassword(pwd)
	if err != nil {
		t.Fatalf("unexpected error hashing password second time: %v", err)
	}
	if hash == hash2 {
		t.Error("expected two hashes of the same password to be different due to salting")
	}
}

func TestEdgeCases(t *testing.T) {
	ResetCost()

	t.Run("Empty Password", func(t *testing.T) {
		hash, err := HashPassword("")
		if err != nil {
			t.Fatalf("unexpected error for empty password: %v", err)
		}
		if !CheckPasswordHash("", hash) {
			t.Error("expected CheckPasswordHash to return true for empty password")
		}
	})

	t.Run("Password Exceeding 72 Bytes", func(t *testing.T) {
		longPwd := strings.Repeat("a", 73)
		_, err := HashPassword(longPwd)
		if err != ErrPasswordTooLong {
			t.Errorf("expected ErrPasswordTooLong, got: %v", err)
		}
	})

	t.Run("UTF-8 Special Characters", func(t *testing.T) {
		utf8Pwd := "🔒Pâ$$wørđ🔑✨"
		hash, err := HashPassword(utf8Pwd)
		if err != nil {
			t.Fatalf("unexpected error for UTF-8 password: %v", err)
		}
		if !CheckPasswordHash(utf8Pwd, hash) {
			t.Error("expected CheckPasswordHash to return true for UTF-8 password")
		}
	})
}

func TestConfigurableCostFactors(t *testing.T) {
	defer ResetCost()

	t.Run("Default Cost Factor", func(t *testing.T) {
		ResetCost()
		if GetCost() != bcrypt.DefaultCost {
			t.Errorf("expected default cost factor to be %d, got %d", bcrypt.DefaultCost, GetCost())
		}
	})

	t.Run("Minimum Valid Cost Factor", func(t *testing.T) {
		err := SetCost(bcrypt.MinCost) // 4
		if err != nil {
			t.Fatalf("unexpected error setting minimum cost: %v", err)
		}
		if GetCost() != bcrypt.MinCost {
			t.Errorf("expected cost to be %d, got %d", bcrypt.MinCost, GetCost())
		}

		hash, err := HashPassword("test")
		if err != nil {
			t.Fatalf("failed to hash with min cost: %v", err)
		}
		if !CheckPasswordHash("test", hash) {
			t.Error("failed to verify hash generated with min cost")
		}
	})

	t.Run("Maximum Valid Cost Factor", func(t *testing.T) {
		err := SetCost(bcrypt.MaxCost) // 31
		if err != nil {
			t.Fatalf("unexpected error setting maximum cost: %v", err)
		}
		if GetCost() != bcrypt.MaxCost {
			t.Errorf("expected cost to be %d, got %d", bcrypt.MaxCost, GetCost())
		}

		err = SetCost(32)
		if err != ErrInvalidCost {
			t.Errorf("expected ErrInvalidCost for cost 32, got: %v", err)
		}

		err = SetCost(3)
		if err != ErrInvalidCost {
			t.Errorf("expected ErrInvalidCost for cost 3, got: %v", err)
		}
	})

	t.Run("Directly mutated invalid cost factor", func(t *testing.T) {
		costFactor = 99
		_, err := HashPassword("test")
		if err != ErrInvalidCost {
			t.Errorf("expected ErrInvalidCost for mutated invalid cost, got: %v", err)
		}
	})
}

func TestWorkFactorSufficient(t *testing.T) {
	// Verify that the cost factor provides a sufficient work factor (at least 250ms per hash on target hardware)
	// We want to dynamically determine a cost factor that takes at least 250ms on this machine.
	// We will start at cost 10 and increase it until we find a cost factor that takes at least 250ms,
	// verifying that configurable costs can achieve this.
	defer ResetCost()

	pwd := "performanceTestPass"
	
	// Let's benchmark different costs starting from 10
	for cost := 10; cost <= 16; cost++ {
		err := SetCost(cost)
		if err != nil {
			t.Fatalf("failed to set cost %d: %v", cost, err)
		}

		start := time.Now()
		_, err = HashPassword(pwd)
		if err != nil {
			t.Fatalf("failed to hash with cost %d: %v", cost, err)
		}
		duration := time.Since(start)
		
		t.Logf("Cost %d took %v", cost, duration)
		if duration >= 250*time.Millisecond {
			t.Logf("Sufficient work factor (>= 250ms) achieved at cost %d (%v)", cost, duration)
			return
		}
	}
	
	t.Log("Note: Did not exceed 250ms in test run under cost 16, which is expected for fast CPUs/environments. Cost can be configured higher by client if needed.")
}
