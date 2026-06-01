// Package hanoi implements the classic Tower of Hanoi problem using optimal recursive algorithms.
// It provides a complete solution for moving n disks from source to destination rod
// while preserving the fundamental constraint that larger disks cannot be placed on smaller ones.
package hanoi

import (
	"errors"
	"fmt"
)

// MaxN represents the upper bound constraint on input n to prevent excessive recursion depth and execution time.
// With O(2^n) time complexity, n=25 results in over 33 million moves, making it a practical upper limit for testing.
const MaxN = 25

var (
	// ErrNegativeInput is returned when n is less than 0.
	ErrNegativeInput = errors.New("input must be a non-negative integer (n >= 0)")

	// ErrInputTooLarge is returned when n exceeds MaxN to protect against excessive execution time.
	ErrInputTooLarge = fmt.Errorf("input exceeds the maximum supported limit of %d", MaxN)
)

// Rod represents the three distinct rods in the Tower of Hanoi problem.
type Rod string

const (
	// Source represents the starting rod containing all disks initially.
	Source Rod = "Source"
	// Auxiliary represents the temporary rod used during the moving process.
	Auxiliary Rod = "Auxiliary"
	// Destination represents the target rod where all disks should end up.
	Destination Rod = "Destination"
)

// Move represents a single disk movement from one rod to another.
// Each move specifies the source rod, destination rod, and the disk number being moved.
type Move struct {
	From Rod    `json:"from"`
	To   Rod    `json:"to"`
	Disk int    `json:"disk"`
}

// String provides a human-readable representation of a move.
func (m Move) String() string {
	return fmt.Sprintf("Move disk %d from %s to %s", m.Disk, m.From, m.To)
}

// Solution represents the complete solution for the Tower of Hanoi problem.
// It contains the sequence of moves and metadata about the solution.
type Solution struct {
	Moves     []Move `json:"moves"`
	NumMoves  int    `json:"num_moves"`
	NumDisks  int    `json:"num_disks"`
}

// Solve computes the optimal solution for the Tower of Hanoi problem with n disks.
//
// Mathematical Foundation:
//   The Tower of Hanoi problem follows the recurrence relation:
//   T(0) = 0 (no moves needed for 0 disks)
//   T(n) = 2*T(n-1) + 1 for n > 0
//   This resolves to T(n) = 2^n - 1 total moves.
//
// Algorithm:
//   1. Move the top n-1 disks from source to auxiliary rod
//   2. Move the largest disk from source to destination rod
//   3. Move the n-1 disks from auxiliary to destination rod
//
// Complexity:
//   - Time Complexity: O(2^n) - each recursive call generates 2^n - 1 moves
//   - Space Complexity: O(n) - maximum recursion depth is n
//
// Constraints:
//   - n must be non-negative and not exceed MaxN (30)
//   - Larger disks cannot be placed on top of smaller disks (enforced by algorithm)
func Solve(n int) (*Solution, error) {
	if n < 0 {
		return nil, ErrNegativeInput
	}
	if n > MaxN {
		return nil, ErrInputTooLarge
	}

	// Handle the trivial case
	if n == 0 {
		return &Solution{
			Moves:    []Move{},
			NumMoves: 0,
			NumDisks: 0,
		}, nil
	}

	var moves []Move
	solveRecursive(n, Source, Destination, Auxiliary, &moves)

	return &Solution{
		Moves:    moves,
		NumMoves: len(moves),
		NumDisks: n,
	}, nil
}

// solveRecursive implements the core recursive algorithm for Tower of Hanoi.
// It moves n disks from source rod to destination rod using auxiliary rod as temporary storage.
//
// Parameters:
//   - n: number of disks to move
//   - from: source rod
//   - to: destination rod
//   - aux: auxiliary rod for temporary storage
//   - moves: slice to accumulate the sequence of moves
func solveRecursive(n int, from, to, aux Rod, moves *[]Move) {
	if n == 1 {
		// Base case: move the single disk directly
		*moves = append(*moves, Move{From: from, To: to, Disk: 1})
		return
	}

	// Step 1: Move top n-1 disks from source to auxiliary rod
	solveRecursive(n-1, from, aux, to, moves)

	// Step 2: Move the largest disk (disk n) from source to destination
	*moves = append(*moves, Move{From: from, To: to, Disk: n})

	// Step 3: Move the n-1 disks from auxiliary to destination rod
	solveRecursive(n-1, aux, to, from, moves)
}

// ValidateSolution verifies that a given solution is mathematically correct.
// It checks that the number of moves equals 2^n - 1 and that the sequence maintains
// the Tower of Hanoi constraints.
func ValidateSolution(solution *Solution) error {
	n := solution.NumDisks
	expectedMoves := (1 << n) - 1 // 2^n - 1

	if solution.NumMoves != expectedMoves {
		return fmt.Errorf("invalid number of moves: expected %d, got %d", expectedMoves, solution.NumMoves)
	}

	if len(solution.Moves) != expectedMoves {
		return fmt.Errorf("moves slice length mismatch: expected %d, got %d", expectedMoves, len(solution.Moves))
	}

	// Additional validation: simulate the moves to ensure no rule violations
	return simulateAndValidate(solution)
}

// Rod state for validation simulation
type rodState struct {
	disks []int // stack of disks (top disk = last element)
}

// simulateAndValidate runs through the entire move sequence to verify no constraints are violated.
func simulateAndValidate(solution *Solution) error {
	n := solution.NumDisks

	// Initialize rod states
	rods := map[Rod]*rodState{
		Source:      {disks: make([]int, n)},
		Auxiliary:   {disks: []int{}},
		Destination: {disks: []int{}},
	}

	// Place all disks on source rod (largest at bottom, smallest at top)
	for i := 0; i < n; i++ {
		rods[Source].disks[i] = n - i
	}

	// Execute each move and validate constraints
	for i, move := range solution.Moves {
		fromRod := rods[move.From]
		toRod := rods[move.To]

		// Check if source rod has any disks
		if len(fromRod.disks) == 0 {
			return fmt.Errorf("move %d: cannot move from empty rod %s", i+1, move.From)
		}

		// Get the top disk from source rod
		topDisk := fromRod.disks[len(fromRod.disks)-1]

		// Verify the move claims to move the correct disk
		if topDisk != move.Disk {
			return fmt.Errorf("move %d: claimed to move disk %d, but top disk is %d", i+1, move.Disk, topDisk)
		}

		// Check constraint: larger disk cannot be placed on smaller disk
		if len(toRod.disks) > 0 && topDisk > toRod.disks[len(toRod.disks)-1] {
			return fmt.Errorf("move %d: cannot place disk %d on top of smaller disk %d", i+1, topDisk, toRod.disks[len(toRod.disks)-1])
		}

		// Execute the move
		fromRod.disks = fromRod.disks[:len(fromRod.disks)-1]
		toRod.disks = append(toRod.disks, topDisk)
	}

	// Verify final state: all disks should be on destination rod in correct order
	destRod := rods[Destination]
	if len(destRod.disks) != n {
		return fmt.Errorf("final validation: destination rod has %d disks, expected %d", len(destRod.disks), n)
	}

	for i := 0; i < n; i++ {
		expectedDisk := n - i
		if destRod.disks[i] != expectedDisk {
			return fmt.Errorf("final validation: disk at position %d is %d, expected %d", i, destRod.disks[i], expectedDisk)
		}
	}

	return nil
}