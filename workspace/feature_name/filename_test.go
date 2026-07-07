package feature_name

import (
    "testing"
    "github.com/dothanhlam/harness-app/password"
)

func TestHashAndVerify(t *testing.T) {
    userPassword := "testpassword123"
    hashedPassword, err := password.Hash(userPassword)
    if err != nil {
        t.Fatalf("Failed to hash password: %v", err)
    }

    // Verify the hashed password
    isCorrect := password.Verify(hashedPassword, userPassword)
    if !isCorrect {
        t.Errorf("Password verification failed! Expected true, got false")
    }
}