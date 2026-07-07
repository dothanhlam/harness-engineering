package bcrypt_js

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "securePassword123"
	hashedPassword, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hashedPassword == "" {
		t.Fatal("Hashed password should not be empty")
	}
}

func TestComparePasswords(t *testing.T) {
	password := "securePassword123"
	hashedPassword, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	err = ComparePasswords(password, hashedPassword)
	if err != nil {
		t.Fatalf("Failed to compare passwords: %v", err)
	}
}