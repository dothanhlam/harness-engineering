- **Secure Password Hashing**: The `password` package offers robust passwor[7D[K
password hashing and verification using the bcrypt algorithm, ensuring stri[4D[K
strict compliance with security bounds.
- **Configurable Cost Factors**: It allows users to configure a cost factor[6D[K
factor between 4 and 31 for hashing. A default cost of 10 is utilized if no[2D[K
not customized.
- **Dynamic Work Factor Assessment**: A test verifying that configurable co[2D[K
costs achieve a sufficient work factor (at least 250ms per hash on target h[1D[K
hardware) has been implemented, starting from the default cost of 10 and in[2D[K
increasing it up to 16 in this scenario. This ensures the security measure'[8D[K
measure's effectiveness across different hardware specifications.
- **Edge Case Handling**: The package handles edge cases effectively, inclu[5D[K
including empty passwords, passwords exceeding the maximum bcrypt limit of [K
72 bytes, and UTF-8 special characters in passwords.
- **Error Management**: It provides clear error messages for invalid cost f[1D[K
factors (`ErrInvalidCost`) and when the password byte length exceeds the bc[2D[K
bcrypt limit (`ErrPasswordTooLong`), avoiding silent truncation or security[8D[K
security vulnerabilities.