# Definition of Done: Factorial Function Implementation

## 1. Functional Requirements
- [ ] Implement the core mathematical logic for calculating the factorial of a non-negative integer ($n!$).
- [ ] Support calculation for $n = 0$ (result must be $1$).
- [ ] Support calculation for $n = 1$ (result must be $1$).
- [ ] Ensure the implementation accurately returns the product of all positive integers less than or equal to $n$.

## 2. Technical Specifications & Error Handling
- [ ] **Algorithm Efficiency:**
    - [ ] Implement with $O(n)$ Time Complexity.
    - [ ] Optimize for $O(1)$ Space Complexity by using an iterative approach to avoid stack overhead.
- [ ] **Data Types & Overflow:**
    - [ ] Use `uint64` for primary implementation.
    - [ ] **Overflow Detection:** Explicitly detect and return an error when $n!$ exceeds `math.MaxUint64` (e.g., for $n > 20$).
    - [ ] **Negative Input:** Reject negative integers with a specific error message or constant.
- [ ] **(Optional) Arbitrary Precision:** 
    - [ ] If required for $n > 20$, provide a secondary implementation or method using `math/big`.

## 3. Workspace & Module Configuration
- [ ] **Module Boundary Fix:** Ensure `workspace/go.mod` is correctly initialized and contains the necessary module path (e.g., `harness/workspace`).
- [ ] **Pathing Resolution:** Resolve the compilation error where the root module fails to find `./workspace/...` as a package. 
    - [ ] Either initialize a `go.work` file in the project root to include `./` and `./workspace`.
    - [ ] Or ensure the build/test commands are executed from the correct module root (`/workspace`).
- [ ] **Dependency Hygiene:** Run `go mod tidy` in the `workspace` directory to ensure `go.sum` is synchronized.

## 4. Code Quality & Standards
- [ ] Code adheres to `go fmt` and `go vet` standards.
- [ ] Public functions are documented following Godoc conventions.
- [ ] Variable naming is idiomatic (e.g., `n`, `result`, `err`).
- [ ] Logic is stateless and safe for concurrent execution.

## 5. Testing & Validation
- [ ] **Unit Tests:**
    - [ ] **Base Cases:** $n=0, n=1$.
    - [ ] **Standard Cases:** $n=5, n=10$.
    - [ ] **Boundary Cases:** Max $n$ before overflow ($n=20$ for `uint64`).
- [ ] **Negative Scenarios:** Assert that negative inputs and overflow conditions return expected errors.
- [ ] **Verified Build Command:** Confirm that `go test ./...` executes successfully from within the `workspace` directory.
- [ ] **Performance Benchmarking:** Include `BenchmarkFactorial` to verify $O(n)$ scaling.

## 6. Documentation & Integration
- [ ] `workspace/factorial/factorial.go`: Core logic.
- [ ] `workspace/factorial/factorial_test.go`: Tests and benchmarks.
- [ ] `workspace/factorial/README.md`: Technical overview and complexity analysis.
- [ ] `workspace/factorial/RELEASE_NOTES.md`: Known constraints (e.g., max $n$ for `uint64`).

## 7. Definitions of Success
- [ ] The code compiles without "directory prefix does not contain main module" errors.
- [ ] All tests pass with 100% coverage on mathematical edge cases.
- [ ] The module structure is clean, following Go's multi-module or workspace best practices.