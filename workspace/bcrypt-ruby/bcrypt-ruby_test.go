package bcrypt_ruby

import (
    "testing"
    "golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
    password := "securePassword123"
    hashedPassword, err := HashPassword(password)
    if err != nil {
        t.Fatalf("Failed to hash password: %v", err)
    }
    if hashedPassword == "" {
        t.Fatal("Hashed password is empty")
    }
}

func TestVerifyPassword(t *testing.T) {
    password := "securePassword123"
    hashedPassword, err := HashPassword(password)
    if err != nil {
        t.Fatalf("Failed to hash password: %v", err)
    }
    err = VerifyPassword(hashedPassword, password)
    if err != nil {
        t.Fatalf("Failed to verify password: %v", err)
    }
}