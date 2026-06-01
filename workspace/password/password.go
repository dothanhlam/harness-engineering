// Package password implements secure password hashing and verification using the bcrypt algorithm.
// It ensures strict compliance with security bounds, configurable cost factors, and UTF-8 handling.
package password

import (
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrPasswordTooLong is returned when the password byte length exceeds the maximum bcrypt limit of 72 bytes.
	ErrPasswordTooLong = errors.New("password length exceeds the maximum bcrypt limit of 72 bytes")

	// ErrInvalidCost is returned when the configured cost factor is outside the valid range of 4 to 31.
	ErrInvalidCost = errors.New("cost factor must be between 4 and 31")
)

// defaultCost factor utilized if cost is not customized.
const defaultCost = bcrypt.DefaultCost // 10

// costFactor is the package-level active cost factor for hashing.
var costFactor = defaultCost

// GetCost returns the current active cost factor.
func GetCost() int {
	return costFactor
}

// SetCost sets the active cost factor for hashing.
// It returns ErrInvalidCost if the cost factor is not within the valid range (Min: 4, Max: 31).
func SetCost(cost int) error {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return ErrInvalidCost
	}
	costFactor = cost
	return nil
}

// ResetCost resets the active cost factor back to the default value (10).
func ResetCost() {
	costFactor = defaultCost
}

// HashPassword generates a secure, salted bcrypt hash of a plain-text password.
// It utilizes the current active cost factor configured in the package.
//
// Security & Limits Warning:
// Bcrypt has an inherent limit of 72 bytes. If the input password byte length exceeds 72 bytes,
// HashPassword will return ErrPasswordTooLong to prevent silent truncation or security vulnerabilities.
// Full UTF-8 support is ensured by validating the raw byte length of the string.
func HashPassword(password string) (string, error) {
	// Bcrypt has a maximum input length of 72 bytes.
	if len(password) > 72 {
		return "", ErrPasswordTooLong
	}

	// Validate cost factor range
	if costFactor < bcrypt.MinCost || costFactor > bcrypt.MaxCost {
		return "", ErrInvalidCost
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), costFactor)
	if err != nil {
		// Ensure no plain-text passwords or credentials leak in error messages.
		return "", fmt.Errorf("failed to generate password hash: %w", err)
	}

	return string(hashedBytes), nil
}

// CheckPasswordHash verifies a plain-text password against a stored bcrypt hash.
// It returns true if the password matches the hash, and false otherwise.
// The comparison is resistant to timing attacks as it uses bcrypt.CompareHashAndPassword.
func CheckPasswordHash(password, hash string) bool {
	// Standard constant-time comparison to prevent timing attacks.
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
