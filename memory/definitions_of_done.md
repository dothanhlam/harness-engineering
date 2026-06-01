# Definition of Done: Fibonacci Function Implementation

## 1. Core Logic & Algorithm
- [ ] **Algorithm Selection:** Implementation of an efficient Fibonacci algorithm (e.g., Iterative for O(n) time/O(1) space or Matrix Exponentiation for O(log n) time).
- [ ] **Data Type Handling:** Use of arbitrary-precision integers (e.g., `math/big` in Go) to prevent overflow for large `n`.
- [ ] **Input Validation:** Strict validation for non-negative integer inputs ($n \ge 0$); return clear error/exception for negative values or non-integer types.
- [ ] **Base Case Integrity:** Explicit handling of $F(0) = 0$ and $F(1) = 1$.

## 2. Technical Architecture & Integration
- [ ] **Interface Definition:** Clean, typed function signature (e.g., `func Fibonacci(n int) (*big.Int, error)`).
- [ ] **Statelessness:** Ensure the core function is side-effect free and deterministic.
- [ ] **Optimization (Caching):** Implementation of memoization if the function is part of a recursive or frequently-called API to ensure $O(1)$ retrieval for previously computed values.
- [ ] **Resource Management:** Bounded execution time and memory usage for extremely large $n$ to prevent stack overflow or heap exhaustion.

## 3. Quality Assurance & Testing
- [ ] **Unit Tests:** 100% path coverage including:
    - [ ] Base cases: $n=0, n=1$.
    - [ ] Standard cases: $n=2, n=10$.
    - [ ] Large cases: $n=92$ (max uint64) and $n > 100$ (big int verification).
- [ ] **Property-Based Testing:** Validation of the property $F(n) = F(n-1) + F(n-2)$ for random $n > 1$.
- [ ] **Performance Benchmarks:** Execution time measurements for $n=1,000$, $n=10,000$, and $n=100,000$ to ensure linear or logarithmic scaling.
- [ ] **Static Analysis:** Zero violations from project-standard linters (e.g., `staticcheck`, `golangci-lint`).

## 4. Documentation & Maintenance
- [ ] **Code Documentation:** Standard-compliant docstrings explaining complexity (Big O notation), constraints, and error conditions.
- [ ] **Example Usage:** Included testable examples or README snippets demonstrating library integration.
- [ ] **Type Safety:** Verified exports and visibility modifiers adhere to the project's encapsulation standards.
