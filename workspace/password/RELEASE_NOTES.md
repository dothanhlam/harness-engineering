- Secure password hashing and verification using the bcrypt algorithm.
- Ensures strict compliance with security bounds, configurable cost factors[7D[K
factors, and UTF-8 handling.
- `ErrPasswordTooLong` returned when password byte length exceeds 72 bytes [K
limit of bcrypt.
- `ErrInvalidCost` returned for invalid cost factor range (4 to 31).
- Default cost factor is `bcrypt.DefaultCost` (10).
- `GetCost`, `SetCost`, and `ResetCost` functions allow dynamic configurati[11D[K
configuration of the active cost factor.
- `HashPassword` generates a secure, salted bcrypt hash using the current a[1D[K
active cost factor. It warns about the inherent 72 bytes limit.
- `CheckPasswordHash` verifies plain-text password against stored bcrypt ha[2D[K
hash, resistant to timing attacks.
- Test cases cover standard hashing, UTF-8 special characters, empty passwo[6D[K
passwords, long passwords exceeding 72 bytes, and configurable cost factors[7D[K
factors.