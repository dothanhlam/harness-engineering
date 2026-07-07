package bcrypt_ruby

import (
    "golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plain text password using bcrypt and returns the hashed password.
func HashPassword(password string) (string, error) {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    return string(hashedPassword), nil
}

// VerifyPassword checks if the provided plain text password matches the hashed password.
func VerifyPassword(hashedPassword, password string) error {
    return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}