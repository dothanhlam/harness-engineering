- Secure password hashing and verification using the bcrypt algorithm
- Ensures strict compliance with security bounds
- Configurable cost factors to adjust work factor per hash
- UTF-8 handling support for special characters in passwords
- `ErrPasswordTooLong` returned when password exceeds 72 bytes limit
- `ErrInvalidCost` returned for invalid cost factor range (4 to 31)
- `GetCost`, `SetCost`, and `ResetCost` functions to manage active cost fac[3D[K
factor