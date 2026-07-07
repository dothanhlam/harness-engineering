package main

import (
	"testing"
)

func TestHexTo25basedConverter(t *testing.T) {
	testCases := []struct {
		hexStr   string
		expected string
	}{
		{"0", "0"},
		{"1", "1"},
		{"10", "10"},
		{"1a", "1j"},
		{"1a3f", "1j1e"},
		{"100", "100"},
		{"ff", "1j0"},
		{"1000", "1000"},
	}

	for _, tc := range testCases {
		result, err := hexTo25basedConverter(tc.hexStr)
		if err != nil {
			t.Errorf("Error converting %s: %v", tc.hexStr, err)
		}
		if result != tc.expected {
			t.Errorf("Expected %s, got %s for %s", tc.expected, result, tc.hexStr)
		}
	}

	// Test invalid input
	invalidInputs := []string{"", "g", "123g", "1a3fz"}
	for _, invalid := range invalidInputs {
		_, err := hexTo25basedConverter(invalid)
		if err == nil {
			t.Errorf("Expected error for invalid input %s, but got none", invalid)
		}
	}
}