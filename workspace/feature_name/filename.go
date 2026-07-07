package feature_name

import (
    "fmt"
    "log"
    "github.com/dothanhlam/harness-app/password"
)

func main() {
    // Example usage of the password hashing and verification functions
    userPassword := "securepassword123"
    hashedPassword, err := password.Hash(userPassword)
    if err != nil {
        log.Fatalf("Failed to hash password: %v", err)
    }
    fmt.Printf("Hashed Password: %s\n", hashedPassword)

    // Verify the hashed password
    isCorrect := password.Verify(hashedPassword, userPassword)
    if isCorrect {
        fmt.Println("Password verification successful!")
    } else {
        fmt.Println("Password verification failed!")
    }
}