# Fibonacci Modular Package

A clean, thread-safe, high-performance modular package implementing arbitrary-precision **Fibonacci** computations in Go.

---

## 1. Core Architecture

The package implements an optimized, production-grade Fibonacci sequence generation library that avoids overflow for large inputs by utilizing Go's native `math/big` package.

### Computation Engine (`pureFibFast`)
Instead of slow $O(n)$ recursion or loop iteration, the core computation utilizes the **Fast Doubling** technique.
Fast Doubling is a form of matrix exponentiation that reduces the number of operations to $O(\log n)$ big integer operations.

The core doubling identities used are:
$$F(2k) = F(k) \times (2F(k+1) - F(k))$$
$$F(2k+1) = F(k+1)^2 + F(k)^2$$

### Memoization Caching
To ensure instant $O(1)$ response times for subsequent queries, a package-level RWMutex (`sync.RWMutex`) is wrapped around a lookup map `fibCache`.
- **Stateless & Read-Optimized:** Read locks are acquired first. If the requested Fibonacci term is cached, a clone is returned immediately without entering write-lock states.
- **Immutability (Defensive Copying):** Go's `math/big` values are mutable. To prevent external users from accidentally corrupting the package cache, the library enforces defensive copying of all retrieved and cached instances.

---

## 2. Technical Features

### Resource Bounds Protection
- **Input Guardrails:** Validates parameter bounds to ensure $n \ge 0$, returning `ErrNegativeInput` for invalid queries.
- **Memory Protection:** Large parameter values are strictly capped at `MaxN = 500,000` via `ErrInputTooLarge` to mitigate CPU denial of service or out-of-memory errors on extreme values.

---

## 3. How to Build & Run Tests

Verify full compliance and zero linter/compiler issues using standard Go tools from the root folder:

### Run Unit & Property-Based Tests
```bash
go test -v ./workspace/fibonacci/...
```

### Run Performance Benchmarks
```bash
go test -bench=. -benchmem ./workspace/fibonacci/...
```
