- Introduce efficient factorial computation with overflow detection in the [K
'factorial' package.
- Implement an iterative approach for `Factorial(n)` with O(n) time complex[7D[K
complexity and O(1) space complexity.
- Add overflow detection and enforce an upper bound (`maxFactorialUint64 = [K
20`) to prevent unhandled overflow or excessive computation.
- Include utility functions: `IsFactorialOverflow(n)` for pre-computation o[1D[K
overflow check, and `MaxSafeFactorial()` to return the maximum safe value o[1D[K
of n.