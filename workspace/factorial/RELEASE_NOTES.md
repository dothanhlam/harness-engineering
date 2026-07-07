* Added `factorial` package for efficient factorial computation with overflow detection
* `Factorial` function computes the factorial of a non-negative integer n with O(n) time and O(1) space complexity
* Includes overflow detection for uint64 and enforces an upper bound to prevent unhandled overflow or excessive computation
* `IsFactorialOverflow` utility function checks if computing factorial(n) would overflow uint64
* `MaxSafeFactorial` utility function returns the maximum value of n for which factorial(n) can be computed without overflow in uint64
* Comprehensive test coverage including base cases, standard cases, boundary cases, negative input, overflow detection, sequential consistency, and benchmarking
* Package documentation and examples provided

> EOF by user