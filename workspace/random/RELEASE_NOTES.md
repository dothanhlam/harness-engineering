- Random package provides thread-safe pseudo-random and cryptographically secure number generation
- Generator struct wraps a PRNG source with mutual exclusion for thread safety
- Global singleton instance of Generator is available for convenience
- IntRange, Float, and Bytes functions generate random values within specified ranges
- ShuffleBytes shuffles a byte slice in-place
- CryptoBytes, CryptoIntRange, and CryptoFloat use cryptographically secure randomness
- Various tests ensure correct operation, error handling, and thread safety
- Chi-squared tests validate uniform distribution of generated values
- Custom seeded instances generate predictable sequences
- Auto-seeding initializes generators with high-entropy sources

> EOF by user