package email_validation

import (
	"strings"
	"testing"
)

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		// --- Positive Test Cases ---
		{"Simple standard email", "user@example.com", true},
		{"Email with subdomain", "user@mail.example.com", true},
		{"Permitted symbols in local-part", "user.name+tag@example.com", true},
		{"Underscore in local-part", "user_name@example.com", true},
		{"Percent in local-part", "user%name@example.com", true},
		{"Dash in local-part", "user-name@example.com", true},
		{"Hyphen in domain", "user@example-domain.com", true},
		{"Single character local-part", "a@example.com", true},
		{"Numeric local-part and domain", "123@456.com", true},

		// --- Negative Test Cases ---
		{"Empty string", "", false},
		{"Missing @ symbol", "userexample.com", false},
		{"Multiple @ symbols", "user@name@example.com", false},
		{"Missing domain", "user@", false},
		{"Missing local-part", "@example.com", false},
		{"Spaces in email", "user name@example.com", false},
		{"Space at start", " user@example.com", false},
		{"Space at end", "user@example.com ", false},
		{"Consecutive dots in local-part", "user..name@example.com", false},
		{"Consecutive dots in domain", "user@example..com", false},
		{"Starting with dot in local-part", ".user@example.com", false},
		{"Ending with dot in local-part", "user.@example.com", false},
		{"Starting with dot in domain", "user@.example.com", false},
		{"Ending with dot in domain", "user@example.com.", false},
		{"TLD suffix too short (1 char)", "user@example.c", false},
		{"TLD suffix missing", "user@example", false},
		{"TLD suffix has digits", "user@example.c12", false},
		{"Special char at start of local-part (+)", "+user@example.com", false},
		{"Special char at end of local-part (+)", "user+@example.com", false},
		{"Special char at start of local-part (_)", "_user@example.com", false},
		{"Special char at end of local-part (_)", "user_@example.com", false},
		{"Special char at start of domain (-)", "user@-example.com", false},
		{"Special char at end of domain (-)", "user@example-.com", false},
		{"Total length exceeds 254 chars", strings.Repeat("a", 245) + "@example.com", false}, // total 257 chars
		{"Total length exactly 254 chars", strings.Repeat("a", 242) + "@example.com", true},  // total 254 chars
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidEmail(tt.email)
			if result != tt.expected {
				t.Errorf("IsValidEmail(%q) = %v; expected %v", tt.email, result, tt.expected)
			}
		})
	}
}

func BenchmarkIsValidEmail(b *testing.B) {
	emails := []string{
		"user@example.com",
		"user.name+tag@example.com",
		"invalid..email@example.com",
		"user@very.long.subdomain.example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, email := range emails {
			_ = IsValidEmail(email)
		}
	}
}
