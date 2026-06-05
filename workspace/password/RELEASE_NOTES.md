- Secure password hashing using bcrypt algorithm with strict compliance to [K
security bounds.
- Configurable cost factors (Min: 4, Max: 31) allow customization of the ha[2D[K
hashing difficulty.
- UTF-8 support ensures handling of special characters in passwords.
- `HashPassword` function generates secure, salted bcrypt hashes with confi[5D[K
configurable cost factor.
- `CheckPasswordHash` verifies plain-text password against stored bcrypt ha[2D[K
hash, resistant to timing attacks.
- Package-level active cost factor tracked via `GetCost`, `SetCost`, and `R[2D[K
`ResetCost`.
- Tests cover edge cases including empty passwords, 72 byte length limit, a[1D[K
and UTF-8 characters.
- Performance benchmarking ensures configurable costs provide sufficient wo[2D[K
work factor (at least 250ms per hash on target hardware).