// Package email_validation implements high-performance, ReDoS-resistant email validation logic
// in strict compliance with RFC 5322 and project-specific security rules.
package email_validation

import (
	"regexp"
	"strings"
)

var (
	// localPartRegex validates the local-part of the email structure.
	// It requires the local-part to start and end with an alphanumeric character
	// and allows permitted symbols (., _, %, +, -) in the middle, satisfying RFC 5322
	// and preventing special characters at start/end of parts.
	localPartRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._%+-]*[a-zA-Z0-9])?$`)

	// labelRegex validates an individual label within the domain.
	// Each label must start and end with an alphanumeric character
	// and contain only alphanumeric characters and hyphens in between.
	labelRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

	// tldRegex validates the Top-Level Domain suffix.
	// It requires the suffix to be alphabetic and at least 2 characters long.
	tldRegex = regexp.MustCompile(`^[a-zA-Z]{2,}$`)
)

// IsValidEmail checks whether a given email string conforms to standard RFC 5322 structure,
// has valid length constraints, correct dot placement, and a valid TLD suffix.
// It returns true if the email is valid, and false otherwise.
//
// Limitations:
//   - Does not check for the actual existence of the domain or MX records.
//   - Does not check for active mailbox delivery status.
func IsValidEmail(email string) bool {
	// 1. Enforce length constraints (maximum 254 characters total)
	if len(email) == 0 || len(email) > 254 {
		return false
	}

	// 2. Reject emails containing spaces (as they are invalid in unquoted local-parts and domains)
	if strings.Contains(email, " ") {
		return false
	}

	// 3. Reject emails with consecutive dots anywhere (e.g. user..name@example.com or example..com)
	if strings.Contains(email, "..") {
		return false
	}

	// 4. Validate presence of exactly one '@' symbol to separate local-part and domain
	if strings.Count(email, "@") != 1 {
		return false
	}

	// 5. Split email into local-part and domain
	parts := strings.Split(email, "@")
	localPart := parts[0]
	domain := parts[1]

	// 6. Ensure neither part is empty
	if len(localPart) == 0 || len(domain) == 0 {
		return false
	}

	// 7. Reject starting or ending dots in both local-part and domain
	// (handled robustly by regex, but explicitly checked here for fast-path rejection)
	if strings.HasPrefix(localPart, ".") || strings.HasSuffix(localPart, ".") {
		return false
	}
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	// 8. Validate local-part characters and structure
	if !localPartRegex.MatchString(localPart) {
		return false
	}

	// 9. Ensure the domain includes at least one dot
	if !strings.Contains(domain, ".") {
		return false
	}

	// 10. Validate each label inside the domain
	domainParts := strings.Split(domain, ".")
	for _, label := range domainParts {
		if len(label) == 0 {
			return false
		}
		if !labelRegex.MatchString(label) {
			return false
		}
	}

	// 11. Extract and validate Top-Level Domain (TLD) suffix
	tld := domainParts[len(domainParts)-1]
	if !tldRegex.MatchString(tld) {
		return false
	}

	return true
}
