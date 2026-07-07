package main

import (
	"fmt"
	"strconv"
)

// hexTo25basedConverter converts a hexadecimal string to a base 25 string.
func hexTo25basedConverter(hexStr string) (string, error) {
	// Validate input
	if hexStr == "" || len(hexStr)%2 != 0 {
		return "", fmt.Errorf("invalid hexadecimal input")
	}

	// Convert hex to decimal
	decimal, err := strconv.ParseInt(hexStr, 16, 64)
	if err != nil {
		return "", err
	}

	// Convert decimal to base 25
	const base = 25
	if decimal == 0 {
		return "0", nil
	}

	var result string
	for decimal > 0 {
		remainder := decimal % base
		result = string('a'+remainder) + result
		decimal /= base
	}

	return result, nil
}

func main() {
	hexStr := "1a3f"
	result, err := hexTo25basedConverter(hexStr)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Base 25:", result)
	}
}