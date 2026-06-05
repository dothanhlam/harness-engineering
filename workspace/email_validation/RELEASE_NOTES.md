- New `email_validation` package for strict RFC 5322 compliant and project-[8D[K
project-specific email validation logic, with ReDoS resistance.
- Single function `IsValidEmail` to check if given string is a valid email [K
address.
- Utilizes regex patterns for local-part, label, and TLD validation.
- Fast-path rejections for empty strings, missing '@', spaces, dots, etc., [K
before full validation.
- Comprehensive test suite covering positive and negative scenarios.
- Benchmarking available for performance evaluation.