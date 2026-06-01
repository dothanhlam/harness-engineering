package gomoku

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

// TestParseCoordinates tests standard alphanumeric coordinate parsing and sanitization.
func TestParseCoordinates(t *testing.T) {
	tests := []struct {
		input    string
		expected Position
		hasError bool
	}{
		{"A1", Position{Row: 0, Col: 0}, false},
		{"a1", Position{Row: 0, Col: 0}, false},
		{"H8", Position{Row: 7, Col: 7}, false},
		{"O15", Position{Row: 14, Col: 14}, false},
		{"  h8  ", Position{Row: 7, Col: 7}, false}, // with whitespace
		{"P1", Position{}, true},                  // invalid col
		{"A0", Position{}, true},                  // invalid row
		{"A16", Position{}, true},                 // invalid row
		{"invalid", Position{}, true},             // junk input
		{"", Position{}, true},                    // empty input
	}

	for _, tc := range tests {
		res, err := ParseCoordinates(tc.input)
		if tc.hasError {
			if err == nil {
				t.Errorf("expected error for coordinate %q, but got nil", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for coordinate %q: %v", tc.input, err)
			}
			if res != tc.expected {
				t.Errorf("for coordinate %q: expected position %+v, got %+v", tc.input, tc.expected, res)
			}
		}
	}
}

// TestPlayMoveAndValidation checks boundary detection and cell occupancy validation.
func TestPlayMoveAndValidation(t *testing.T) {
	g := NewGame()

	// Normal move
	pos, err := g.PlayMove("H8")
	if err != nil {
		t.Fatalf("unexpected error playing H8: %v", err)
	}
	if pos.Row != 7 || pos.Col != 7 {
		t.Errorf("expected coordinate H8 to map to (7,7), got %+v", pos)
	}

	// Double play on same coordinate (occupied cell check)
	err = g.PlayMoveIndices(7, 7)
	if err != ErrCellOccupied {
		t.Errorf("expected ErrCellOccupied, got %v", err)
	}

	// Out of bounds check
	err = g.PlayMoveIndices(-1, 0)
	if err != ErrOutOfBounds {
		t.Errorf("expected ErrOutOfBounds, got %v", err)
	}
	err = g.PlayMoveIndices(0, 15)
	if err != ErrOutOfBounds {
		t.Errorf("expected ErrOutOfBounds, got %v", err)
	}
}

// TestHorizontalWin checks win detection on horizontal alignment.
func TestHorizontalWin(t *testing.T) {
	g := NewGame()
	// Row 5: H5, I5, J5, K5, L5
	moves := []string{"H5", "H6", "I5", "I6", "J5", "J6", "K5", "K6", "L5"}
	// Plays:
	// Black: H5, I5, J5, K5, L5 (5 in a row!)
	// White: H6, I6, J6, K6
	for i, m := range moves {
		_, err := g.PlayMove(m)
		if err != nil {
			t.Fatalf("unexpected error on move %d (%s): %v", i, m, err)
		}
	}

	if g.GetStatus() != Won {
		t.Error("expected game to be Won horizontally")
	}
	if g.GetWinner() != Black {
		t.Errorf("expected Black to be the winner, got %v", g.GetWinner())
	}
}

// TestVerticalWin checks win detection on vertical alignment.
func TestVerticalWin(t *testing.T) {
	g := NewGame()
	// Col 2 (C): C1, C2, C3, C4, C5
	moves := []string{"C1", "D1", "C2", "D2", "C3", "D3", "C4", "D4", "C5"}
	for _, m := range moves {
		if _, err := g.PlayMove(m); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if g.GetStatus() != Won {
		t.Error("expected game to be Won vertically")
	}
	if g.GetWinner() != Black {
		t.Errorf("expected Black to be winner, got %v", g.GetWinner())
	}
}

// TestDiagonalWin checks win detection on diagonal alignments.
func TestDiagonalWin(t *testing.T) {
	t.Run("DiagonalDownRight", func(t *testing.T) {
		g := NewGame()
		// A1, B2, C3, D4, E5
		moves := []string{"A1", "A2", "B2", "B3", "C3", "C4", "D4", "D5", "E5"}
		for _, m := range moves {
			if _, err := g.PlayMove(m); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}
		if g.GetStatus() != Won {
			t.Error("expected game to be Won diagonally down-right")
		}
	})

	t.Run("DiagonalDownLeft", func(t *testing.T) {
		g := NewGame()
		// E1, D2, C3, B4, A5
		moves := []string{"E1", "E2", "D2", "D3", "C3", "C4", "B4", "B5", "A5"}
		for _, m := range moves {
			if _, err := g.PlayMove(m); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}
		if g.GetStatus() != Won {
			t.Error("expected game to be Won diagonally down-left")
		}
	})
}

// TestOverlineScenario verifies that an overline (6 or more stones) does not trigger a win.
func TestOverlineScenario(t *testing.T) {
	// Scenario: Black places 6 stones in a row.
	// In Gomoku, exactly 5 stones win. An overline of 6 should not be detected as a win.
	g := NewGame()

	// Row 0: A1, B1, C1, D1, E1, F1
	// We will manually place Black stones to construct an overline, ignoring player turns
	// by directly setting the board and checking win at F1.
	g.board[0][0] = Black
	g.board[0][1] = Black
	g.board[0][2] = Black
	g.board[0][3] = Black
	g.board[0][4] = Black
	g.board[0][5] = Black // F1 is the 6th

	if g.hasWonAt(Position{Row: 0, Col: 5}, Black) {
		t.Error("expected overline of 6 stones to NOT trigger a win")
	}

	// But a subset of exactly 5 should trigger a win
	g2 := NewGame()
	g2.board[0][0] = Black
	g2.board[0][1] = Black
	g2.board[0][2] = Black
	g2.board[0][3] = Black
	g2.board[0][4] = Black

	if !g2.hasWonAt(Position{Row: 0, Col: 4}, Black) {
		t.Error("expected exactly 5 stones to trigger a win")
	}
}

// TestUndo verifies rolling back moves completely restores previous state.
func TestUndo(t *testing.T) {
	g := NewGame()

	// Play a few moves
	if _, err := g.PlayMove("H8"); err != nil {
		t.Fatal(err)
	}
	if _, err := g.PlayMove("H9"); err != nil {
		t.Fatal(err)
	}

	// Undo the second move (H9)
	if err := g.Undo(); err != nil {
		t.Fatalf("unexpected undo error: %v", err)
	}

	// Board cell should be Empty
	if g.GetBoard()[8][7] != Empty {
		t.Error("expected H9 to be Empty after undo")
	}

	// Turn should have reverted to White
	if g.GetCurrentTurn() != White {
		t.Errorf("expected current turn to revert to White, got %v", g.GetCurrentTurn())
	}

	// History should only have 1 move
	history := g.GetHistory()
	if len(history) != 1 {
		t.Errorf("expected history length 1, got %d", len(history))
	}
	if history[0].Pos.Row != 7 || history[0].Pos.Col != 7 {
		t.Errorf("expected first history move to be H8, got %+v", history[0])
	}

	// lastMove should point to H8
	last := g.GetLastMove()
	if last == nil || last.Row != 7 || last.Col != 7 {
		t.Errorf("expected lastMove to be H8, got %+v", last)
	}

	// Undo first move (H8)
	if err := g.Undo(); err != nil {
		t.Fatal(err)
	}

	// lastMove should be nil
	if g.GetLastMove() != nil {
		t.Errorf("expected lastMove to be nil, got %+v", g.GetLastMove())
	}

	// No moves left to undo
	if err := g.Undo(); err != ErrNoMoves {
		t.Errorf("expected ErrNoMoves, got %v", err)
	}
}

// TestConcurrencySafety runs multiple goroutines simulating concurrent move queries.
func TestConcurrencySafety(t *testing.T) {
	g := NewGame()
	var wg sync.WaitGroup

	// Run concurrent reads and a safe sequence of writes
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = g.GetBoard()
			_ = g.BoardString()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = g.GetCurrentTurn()
			_ = g.GetStatus()
		}
	}()

	go func() {
		defer wg.Done()
		// Safe sequential moves
		for r := 0; r < 5; r++ {
			_ = g.PlayMoveIndices(r, 0)
		}
	}()

	wg.Wait()
}

// TestIntegrationSimulatedGame plays a full game to win and checks draw condition.
func TestIntegrationSimulatedGame(t *testing.T) {
	t.Run("GameWinSimulated", func(t *testing.T) {
		g := NewGame()
		// Standard game progression leading to Black win
		moves := []string{
			"H8", "H9",
			"I8", "I9",
			"J8", "J9",
			"K8", "K9",
			"L8", // Black wins!
		}
		for _, m := range moves {
			if _, err := g.PlayMove(m); err != nil {
				t.Fatalf("unexpected error playing %s: %v", m, err)
			}
		}

		if g.GetStatus() != Won {
			t.Error("expected status to be Won")
		}
		if g.GetWinner() != Black {
			t.Errorf("expected Winner to be Black, got %v", g.GetWinner())
		}
	})

	t.Run("InteractivePlayQuit", func(t *testing.T) {
		g := NewGame()
		in := strings.NewReader("quit\n")
		var out bytes.Buffer

		err := g.InteractivePlay(in, &out)
		if err != nil {
			t.Fatalf("unexpected InteractivePlay error: %v", err)
		}

		outStr := out.String()
		if !strings.Contains(outStr, "Adios! Thanks for playing.") {
			t.Errorf("expected quit message in output, got %q", outStr)
		}
	})
}

// BenchmarkPlayMoveAndCheckWin benchmarks the placement of stones and win checking.
func BenchmarkPlayMoveAndCheckWin(b *testing.B) {
	for i := 0; i < b.N; i++ {
		g := NewGame()
		// Make 10 moves sequentially and check win (done automatically inside PlayMoveIndices)
		for step := 0; step < 10; step++ {
			_ = g.PlayMoveIndices(step, step)
		}
	}
}
