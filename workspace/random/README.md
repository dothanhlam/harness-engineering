# Random Number Generator Module

A high-performance, thread-safe, and self-contained Go package providing both **Pseudo-Random Number Generation (PRNG)** and **Cryptographically Secure Random Number Generation (CSPRNG)** capabilities.

## Quick Start

Import the package using:
```go
import "harness-engineering/workspace/random"
```

### Basic PRNG Usage

Use the global thread-safe generator instance for simulation, gaming, or non-security-critical tasks:

```go
// 1. Generate an integer within range [min, max] inclusive
val, err := random.IntRange(1, 100)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Random Integer: %d\n", val)

// 2. Generate a floating-point number in [0.0, 1.0)
fVal := random.Float()
fmt.Printf("Random Float: %f\n", fVal)

// 3. Generate a random byte slice
bytes, err := random.Bytes(16)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Random Bytes: %x\n", bytes)

// 4. Shuffle a slice of bytes in-place
data := []byte("hello world")
random.ShuffleBytes(data)
fmt.Printf("Shuffled: %s\n", data)
```

### Custom-Seeded Local Instance

For predictable sequences (e.g., repeatable game sessions or simulations), instantiate custom-seeded generators:

```go
// Create generator with seed1 and seed2
gen := random.NewGenerator(12345, 67890)

// Subsequent calls are deterministic and predictable
val, _ := gen.IntRange(0, 1000)
```

### CSPRNG (Cryptographically Secure) Usage

For security-sensitive work (e.g., generating salts, API keys, tokens, session IDs), use the cryptographically secure functions:

```go
// 1. Generate secure random byte slices
salt, err := random.CryptoBytes(32)
if err != nil {
    log.Fatalf("Entropy source failed: %v", err)
}

// 2. Generate secure random integer in [min, max] inclusive
secureInt, err := random.CryptoIntRange(100000, 999999)
if err != nil {
    log.Fatalf("Entropy source failed: %v", err)
}

// 3. Generate secure random float in [0.0, 1.0)
secureFloat, err := random.CryptoFloat()
if err != nil {
    log.Fatalf("Entropy source failed: %v", err)
}
```

---

## CSPRNG vs PRNG Usage Guide

| Characteristic | Pseudo-Random (PRNG) | Cryptographically Secure (CSPRNG) |
| :--- | :--- | :--- |
| **Engine** | `math/rand/v2` (PCG source) | `crypto/rand` (System entropy) |
| **Security** | **Predictable.** Do NOT use for keys, salts, or passwords. | **Unpredictable.** Suitable for all cryptographic tasks. |
| **Performance** | Sub-microsecond latency (extremely fast). | Higher latency (involves system calls). |
| **Reproducibility** | Supports custom seed for repeatable outputs. | Fully non-repeatable. |
| **Use Cases** | Simulations, shuffling game decks, UI effects. | Password hashing salts, session tokens, keys. |

---

## Verification and Quality Checks

Run the test suite, including race detection and statistical validation:
```bash
go test -v -race ./workspace/random/...
```

Run latency benchmarks:
```bash
go test -bench=. ./workspace/random/...
```

---

## Architecture Note: The "Clear Loop" Context Reset

If the developer agent (`agy`) fails its QA retries, the Harness pipeline activates the **Delegation Protocol** and has the BA Agent rewrite the `definitions_of_done.md`. 

To ensure the Dev agent starts with a perfectly **clear loop** on its next attempt:
* The orchestrator performs a hard context reset by deleting the local state directories (`.antigravitycli` and `.claude`).
* This prevents context pollution, ensuring the agent doesn't "remember" its previous failed code generation attempts or the outdated requirements.
