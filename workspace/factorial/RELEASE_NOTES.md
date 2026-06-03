- New package `factorial` provides an efficient and robust way to compute f[1D[K
factorial values with overflow detection.
- The `Factorial` function computes the factorial of a non-negative integer[7D[K
integer `n`, using an iterative approach for O(n) time complexity and O(1) [K
space complexity.
- It includes upper bound enforcement and overflow checks, preventing unhan[5D[K
unhandled overflow or excessive computation.
- The package also includes utility functions like `IsFactorialOverflow` fo[2D[K
for pre-validation without performing the actual computation and `MaxSafeFa[10D[K
`MaxSafeFactorial` to return the maximum safe value of n.
- Comprehensive test suite covers base cases, standard computations, bounda[6D[K
boundary cases, negative input handling, overflow detection, utility functi[6D[K
function testing, sequential consistency verification, and benchmarking.