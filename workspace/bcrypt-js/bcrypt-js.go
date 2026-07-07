package bcrypt_js

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword takes a plain-text password and returns the hashed password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// ComparePasswords takes a plain-text password and a hashed password and returns true if they match.
func ComparePasswords(plainTextPassword, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainTextPassword))
}