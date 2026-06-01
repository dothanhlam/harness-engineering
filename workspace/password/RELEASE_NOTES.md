# Release Note - Password Package

## New Features

- Added support for configurable bcrypt cost factors, allowing clients to a[1D[K
adjust the computational difficulty based on their hardware capabilities.
- Introduced `HashPassword` and `CheckPasswordHash` functions for secure pa[2D[K
password hashing and verification.

## Improvements

- Updated package to ensure strict compliance with security bounds and UTF-[4D[K
UTF-8 handling.
- Implemented checks for maximum password length (72 bytes) and valid cost [K
factor range (4 to 31).
- Enhanced testing coverage, including edge cases, configurable costs, and [K
performance benchmarking.