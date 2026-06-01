package hanoi

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// TestSolveBaseAndStandardCases tests base cases and standard cases as required.
func TestSolveBaseAndStandardCases(t *testing.T) {
	tests := []struct {
		n             int
		expectedMoves int
		name          string
	}{
		{0, 0, "zero_disks"},
		{1, 1, "one_disk"},
		{2, 3, "two_disks"},
		{3, 7, "three_disks"},
		{4, 15, "four_disks"},
		{5, 31, "five_disks"},
		{10, 1023, "ten_disks"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			solution, err := Solve(tc.n)
			if err != nil {
				t.Fatalf("unexpected error for n=%d: %v", tc.n, err)
			}

			if solution.NumMoves != tc.expectedMoves {
				t.Errorf("expected %d moves for n=%d, got %d", tc.expectedMoves, tc.n, solution.NumMoves)
			}

			if solution.NumDisks != tc.n {
				t.Errorf("expected NumDisks=%d, got %d", tc.n, solution.NumDisks)
			}

			if len(solution.Moves) != tc.expectedMoves {
				t.Errorf("expected %d moves in slice for n=%d, got %d", tc.expectedMoves, tc.n, len(solution.Moves))
			}

			// Validate the solution is mathematically correct
			if err := ValidateSolution(solution); err != nil {
				t.Errorf("solution validation failed for n=%d: %v", tc.n, err)
			}
		})
	}
}

// TestMathematicalVerification explicitly tests the 2^n - 1 formula for various n values.
func TestMathematicalVerification(t *testing.T) {
	testCases := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15, 20}

	for _, n := range testCases {
		t.Run(fmt.Sprintf("n_%d", n), func(t *testing.T) {
			solution, err := Solve(n)
			if err != nil {
				t.Fatalf("unexpected error for n=%d: %v", n, err)
			}

			var expectedMoves int
			if n == 0 {
				expectedMoves = 0
			} else {
				expectedMoves = (1 << n) - 1 // 2^n - 1
			}

			if solution.NumMoves != expectedMoves {
				t.Errorf("mathematical verification failed for n=%d: expected %d moves (2^%d - 1), got %d",
					n, expectedMoves, n, solution.NumMoves)
			}

			if len(solution.Moves) != expectedMoves {
				t.Errorf("moves slice length verification failed for n=%d: expected %d, got %d",
					n, expectedMoves, len(solution.Moves))
			}
		})
	}
}

// TestSolveInputValidation verifies input validation guardrails.
func TestSolveInputValidation(t *testing.T) {
	t.Run("Negative Inputs", func(t *testing.T) {
		solution, err := Solve(-1)
		if err != ErrNegativeInput {
			t.Errorf("expected ErrNegativeInput, got: %v", err)
		}
		if solution != nil {
			t.Errorf("expected nil solution on error, got: %v", solution)
		}

		_, err = Solve(-100)
		if err != ErrNegativeInput {
			t.Errorf("expected ErrNegativeInput, got: %v", err)
		}
	})

	t.Run("Too Large Inputs", func(t *testing.T) {
		solution, err := Solve(MaxN + 1)
		if err != ErrInputTooLarge {
			t.Errorf("expected ErrInputTooLarge, got: %v", err)
		}
		if solution != nil {
			t.Errorf("expected nil solution on error, got: %v", solution)
		}

		_, err = Solve(100) // Well above MaxN
		if err != ErrInputTooLarge {
			t.Errorf("expected ErrInputTooLarge, got: %v", err)
		}
	})
}

// TestMoveSequenceProperties validates algorithmic properties of the move sequence.
func TestMoveSequenceProperties(t *testing.T) {
	testCases := []int{1, 2, 3, 4, 5}

	for _, n := range testCases {
		t.Run(fmt.Sprintf("n_%d_properties", n), func(t *testing.T) {
			solution, err := Solve(n)
			if err != nil {
				t.Fatalf("unexpected error for n=%d: %v", n, err)
			}

			// Property 1: First move should be disk 1
			if len(solution.Moves) > 0 && solution.Moves[0].Disk != 1 {
				t.Errorf("first move should be disk 1, got disk %d", solution.Moves[0].Disk)
			}

			// Property 2: Last move should move to Destination
			if len(solution.Moves) > 0 && solution.Moves[len(solution.Moves)-1].To != Destination {
				t.Errorf("last move should go to Destination, got %s", solution.Moves[len(solution.Moves)-1].To)
			}

			// Property 3: For n > 1, largest disk (disk n) should move exactly once
			if n > 1 {
				largestDiskMoves := 0
				for _, move := range solution.Moves {
					if move.Disk == n {
						largestDiskMoves++
					}
				}
				if largestDiskMoves != 1 {
					t.Errorf("disk %d (largest) should move exactly once, moved %d times", n, largestDiskMoves)
				}
			}
		})
	}
}

// TestStateValidation ensures no rule violations occur during execution.
func TestStateValidation(t *testing.T) {
	testCases := []int{1, 2, 3, 4, 5, 6, 7}

	for _, n := range testCases {
		t.Run(fmt.Sprintf("state_validation_n_%d", n), func(t *testing.T) {
			solution, err := Solve(n)
			if err != nil {
				t.Fatalf("unexpected error for n=%d: %v", n, err)
			}

			// Use the built-in ValidateSolution which simulates the entire sequence
			if err := ValidateSolution(solution); err != nil {
				t.Errorf("state validation failed for n=%d: %v", n, err)
			}
		})
	}
}

// TestMoveStringRepresentation tests the string formatting of moves.
func TestMoveStringRepresentation(t *testing.T) {
	move := Move{From: Source, To: Destination, Disk: 3}
	expected := "Move disk 3 from Source to Destination"
	if move.String() != expected {
		t.Errorf("expected %q, got %q", expected, move.String())
	}
}

// TestRodConstants ensures rod constants are properly defined.
func TestRodConstants(t *testing.T) {
	if Source != "Source" {
		t.Errorf("Source constant incorrect: got %q", Source)
	}
	if Auxiliary != "Auxiliary" {
		t.Errorf("Auxiliary constant incorrect: got %q", Auxiliary)
	}
	if Destination != "Destination" {
		t.Errorf("Destination constant incorrect: got %q", Destination)
	}
}

// TestSolveRandomizedProperties validates algorithmic properties for random inputs.
func TestSolveRandomizedProperties(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Run 50 randomized property validations
	for i := 0; i < 50; i++ {
		n := rng.Intn(15) + 1 // n in range [1, 15] to keep tests fast

		solution, err := Solve(n)
		if err != nil {
			t.Fatalf("unexpected error for n=%d: %v", n, err)
		}

		// Property: number of moves should be 2^n - 1
		expectedMoves := (1 << n) - 1
		if solution.NumMoves != expectedMoves {
			t.Errorf("property violation for n=%d: expected %d moves, got %d", n, expectedMoves, solution.NumMoves)
		}

		// Property: solution should be valid
		if err := ValidateSolution(solution); err != nil {
			t.Errorf("validation failed for random n=%d: %v", n, err)
		}
	}
}

// TestEdgeCaseBoundary tests the boundary at MaxN.
func TestEdgeCaseBoundary(t *testing.T) {
	t.Run("MaxN_Boundary", func(t *testing.T) {
		// Test just below MaxN to ensure it still works
		testN := MaxN - 2 // Test n=23 instead of n=25 to keep test reasonable
		solution, err := Solve(testN)
		if err != nil {
			t.Fatalf("unexpected error for n=%d (near MaxN): %v", testN, err)
		}

		expectedMoves := (1 << testN) - 1
		if solution.NumMoves != expectedMoves {
			t.Errorf("boundary test failed: expected %d moves for n=%d, got %d", expectedMoves, testN, solution.NumMoves)
		}
	})
}

// ExampleSolve demonstrates basic usage of the Solve function.
func ExampleSolve() {
	solution, err := Solve(3)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Solution for 3 disks requires %d moves:\n", solution.NumMoves)
	for i, move := range solution.Moves {
		fmt.Printf("%d. %s\n", i+1, move.String())
	}

	// Output:
	// Solution for 3 disks requires 7 moves:
	// 1. Move disk 1 from Source to Destination
	// 2. Move disk 2 from Source to Auxiliary
	// 3. Move disk 1 from Destination to Auxiliary
	// 4. Move disk 3 from Source to Destination
	// 5. Move disk 1 from Auxiliary to Source
	// 6. Move disk 2 from Auxiliary to Destination
	// 7. Move disk 1 from Source to Destination
}

// Performance benchmarks as required by the specifications.

// BenchmarkSolve10 benchmarks the algorithm for n=10 (1023 moves).
func BenchmarkSolve10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := Solve(10)
		if err != nil {
			b.Fatalf("benchmark error: %v", err)
		}
	}
}

// BenchmarkSolve20 benchmarks the algorithm for n=20 (1,048,575 moves).
func BenchmarkSolve20(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := Solve(20)
		if err != nil {
			b.Fatalf("benchmark error: %v", err)
		}
	}
}

// BenchmarkSolve25 benchmarks the algorithm for n=25 (33,554,431 moves).
func BenchmarkSolve25(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := Solve(25)
		if err != nil {
			b.Fatalf("benchmark error: %v", err)
		}
	}
}

// BenchmarkValidateSolution benchmarks the validation function.
func BenchmarkValidateSolution(b *testing.B) {
	solution, _ := Solve(15) // Pre-generate a solution

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ValidateSolution(solution)
		if err != nil {
			b.Fatalf("validation error: %v", err)
		}
	}
}