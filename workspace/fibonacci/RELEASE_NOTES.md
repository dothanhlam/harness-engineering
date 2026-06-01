- **Fibonacci Sequence Computation**: The package implements high-performan[14D[K
high-performance, arbitrary-precision Fibonacci sequence computations. It u[1D[K
uses an optimized fast doubling algorithm with O(log n) time complexity and[3D[K
and memoization caching for O(1) retrieval of previously calculated terms.
- **Input Validation**: Ensures the input 'n' is a non-negative integer (n [K
>= 0), preventing resource exhaustion by capping it at MaxN (500,000).
- **Thread Safety**: The cache used for memoization is protected against co[2D[K
concurrent access, making the package safe to use in multi-threaded environ[7D[K
environments.
- **Extensive Testing**: Includes tests covering base and standard cases, l[1D[K
large Fibonacci numbers, input validation, property-based validations, and [K
immutability of returned values.