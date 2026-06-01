- **Feature Introduction**: Introducing the 'fibonacci' package which provi[5D[K
provides high-performance, arbitrary-precision calculations for the Fibonac[7D[K
Fibonacci sequence. The feature leverages an optimized fast doubling algori[6D[K
algorithm, achieving O(log n) time complexity and utilizing thread-safe mem[3D[K
memoization caching for O(1) retrieval of previously computed terms.
- **Upper Limit Set**: To prevent resource exhaustion (both heap and CPU), [K
a maximum input constraint (`MaxN`) of 500000 is implemented. This safeguar[8D[K
safeguard ensures the system's stability by not allowing inputs larger than[4D[K
than this limit.
- **Error Handling**: The package includes robust error handling for negati[6D[K
negative or excessively large inputs, returning `ErrNegativeInput` or `ErrI[5D[K
`ErrInputTooLarge` respectively to maintain functionality integrity.
- **Cache Implementation**: Employing a memoization cache (`fibCache`) prot[4D[K
protects against redundant calculations and ensures efficient retrieval of [K
previously calculated Fibonacci numbers. This cache is protected by a mutua[5D[K
mutual exclusion lock (`cacheMu`) for thread safety, making it suitable for[3D[K
for concurrent applications.
- **Functionality Verification**: A suite of tests including base cases, la[2D[K
large case validations, input validation checks, property-based testing for[3D[K
for recurrence relation verification, and immutability assurance ensures th[2D[K
the accuracy and reliability of the feature.