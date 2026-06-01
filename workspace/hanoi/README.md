# Tower of Hanoi Package

A high-performance implementation of the classic Tower of Hanoi problem in Go, featuring optimal recursive algorithms, comprehensive validation, and complete mathematical verification.

## Overview

The Tower of Hanoi is a classic mathematical puzzle consisting of three rods and a number of disks of different sizes that can be stacked on any rod. The puzzle starts with all disks stacked in ascending order of size on one rod (source), with the smallest disk on top. The objective is to move the entire stack to another rod (destination), following these rules:

1. Only one disk can be moved at a time
2. Each move consists of taking the top disk from one of the stacks and placing it on top of another stack
3. No disk may be placed on top of a smaller disk

## Algorithm

This implementation uses the optimal recursive algorithm with the following approach:

### Recursive Strategy
```
HanoiSolve(n, source, destination, auxiliary):
  if n == 1:
    move disk 1 from source to destination
  else:
    HanoiSolve(n-1, source, auxiliary, destination)
    move disk n from source to destination  
    HanoiSolve(n-1, auxiliary, destination, source)
```

### Mathematical Foundation
- **Recurrence Relation**: T(n) = 2*T(n-1) + 1, with T(1) = 1
- **Closed Form**: T(n) = 2^n - 1
- **Proof**: The optimal solution always requires exactly 2^n - 1 moves

## Complexity Analysis

| Metric | Complexity | Explanation |
|--------|------------|-------------|
| **Time Complexity** | O(2^n) | Each recursive call generates 2^n - 1 moves |
| **Space Complexity** | O(n) | Maximum recursion depth equals number of disks |
| **Move Count** | 2^n - 1 | Mathematically proven minimum number of moves |

### Performance Characteristics

| Disks (n) | Moves Required | Approximate Time |
|-----------|----------------|------------------|
| 10 | 1,023 | < 1ms |
| 15 | 32,767 | < 10ms |
| 20 | 1,048,575 | < 100ms |
| 25 | 33,554,431 | < 1s |

**Note**: Times are approximate and depend on hardware. The implementation is optimized for moves up to n=25.

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "github.com/dothanhlam/harness-app/workspace/hanoi"
)

func main() {
    // Solve Tower of Hanoi for 3 disks
    solution, err := hanoi.Solve(3)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Solution for %d disks requires %d moves:\n", 
               solution.NumDisks, solution.NumMoves)
    
    for i, move := range solution.Moves {
        fmt.Printf("%d. %s\n", i+1, move.String())
    }
}
```

### Expected Output
```
Solution for 3 disks requires 7 moves:
1. Move disk 1 from Source to Destination
2. Move disk 2 from Source to Auxiliary  
3. Move disk 1 from Destination to Auxiliary
4. Move disk 3 from Source to Destination
5. Move disk 1 from Auxiliary to Source
6. Move disk 2 from Auxiliary to Destination
7. Move disk 1 from Source to Destination
```

### Advanced Usage with Validation

```go
solution, err := hanoi.Solve(5)
if err != nil {
    log.Fatal(err)
}

// Validate the solution mathematically
if err := hanoi.ValidateSolution(solution); err != nil {
    log.Fatalf("Solution validation failed: %v", err)
}

fmt.Printf("✅ Solution validated: %d moves for %d disks\n", 
           solution.NumMoves, solution.NumDisks)
```

## API Reference

### Core Functions

#### `Solve(n int) (*Solution, error)`
Computes the optimal solution for n disks.

**Parameters:**
- `n`: Number of disks (0 ≤ n ≤ 30)

**Returns:**
- `*Solution`: Complete solution with move sequence
- `error`: Input validation error if any

#### `ValidateSolution(solution *Solution) error`  
Validates a solution by simulating the move sequence and verifying constraints.

**Parameters:**
- `solution`: Solution to validate

**Returns:**
- `error`: Validation error if constraints are violated, nil if valid

### Data Structures

#### `Solution`
```go
type Solution struct {
    Moves     []Move `json:"moves"`      // Sequence of moves
    NumMoves  int    `json:"num_moves"`  // Total number of moves  
    NumDisks  int    `json:"num_disks"`  // Number of disks solved
}
```

#### `Move`
```go
type Move struct {
    From Rod `json:"from"`  // Source rod
    To   Rod `json:"to"`    // Destination rod  
    Disk int `json:"disk"`  // Disk number being moved
}
```

#### `Rod`
```go
type Rod string

const (
    Source      Rod = "Source"      // Starting rod
    Auxiliary   Rod = "Auxiliary"   // Temporary rod
    Destination Rod = "Destination" // Target rod
)
```

## Testing

### Run Unit Tests
```bash
cd workspace/hanoi
go test -v
```

### Run Performance Benchmarks
```bash
cd workspace/hanoi  
go test -bench=.
```

### Test Coverage
```bash
cd workspace/hanoi
go test -cover
```

### Specific Test Categories
```bash
# Test mathematical verification
go test -run TestMathematicalVerification -v

# Test input validation  
go test -run TestSolveInputValidation -v

# Test state validation
go test -run TestStateValidation -v

# Benchmark different disk counts
go test -bench=BenchmarkSolve10 -v
go test -bench=BenchmarkSolve20 -v
```

## Input Constraints

| Constraint | Value | Reason |
|------------|-------|--------|
| **Minimum n** | 0 | No-op case, empty solution |
| **Maximum n** | 25 | Prevents excessive execution time (2^25 ≈ 33 million moves) |
| **Negative n** | Not allowed | Mathematically undefined |

For n > 25, the function returns `ErrInputTooLarge` to prevent resource exhaustion.

## Implementation Features

### ✅ Functional Requirements
- [x] Core recursive algorithm implementation
- [x] Dynamic input support (0 ≤ n ≤ 25)
- [x] Three distinct rods (Source, Auxiliary, Destination)
- [x] Constraint preservation (larger disk never on smaller)
- [x] Deterministic optimal move sequence

### ✅ Performance Characteristics
- [x] Time Complexity: O(2^n) verified
- [x] Space Complexity: O(n) stack depth
- [x] Move Count: Exactly 2^n - 1 moves

### ✅ Quality Assurance
- [x] Go standard formatting (`go fmt`)
- [x] Godoc-compliant documentation
- [x] Comprehensive input validation
- [x] Zero global state (thread-safe)
- [x] Idiomatic Go data structures

### ✅ Testing Coverage
- [x] Unit tests for n=1,2,3 (and beyond)
- [x] Mathematical verification (2^n - 1 formula)
- [x] State validation (no rule violations)
- [x] Performance benchmarks (n=10, 20, 25)
- [x] Property-based testing
- [x] Edge case validation

## License

This implementation is part of the harness-engineering project and follows the same licensing terms.