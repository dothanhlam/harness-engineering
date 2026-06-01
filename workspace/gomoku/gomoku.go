// Package gomoku implements a thread-safe, high-performance Gomoku (Go-moku)
// game engine and CLI renderer compliant with project-specific definitions of done.
package gomoku

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// BoardSize represents the standard Gomoku board size (15x15).
const BoardSize = 15

// Stone represents the type of player stone on the board.
type Stone int

const (
	Empty Stone = iota
	Black // Player 1 (●)
	White // Player 2 (○)
)

// GameStatus represents the current state of the game.
type GameStatus int

const (
	Active GameStatus = iota
	Won
	Draw
)

// Position represents a 0-indexed row and col coordinate.
type Position struct {
	Row int
	Col int
}

// Move represents a single move in the game.
type Move struct {
	Player Stone
	Pos    Position
}

var (
	// Standard alphanumeric coordinate regex (A1 to O15, case-insensitive)
	coordRegex = regexp.MustCompile(`^(?i)([A-O])(1[0-5]|[1-9])$`)

	ErrOutOfBounds  = errors.New("coordinates out of bounds")
	ErrCellOccupied = errors.New("cell is already occupied")
	ErrGameOver     = errors.New("game has already ended")
	ErrNoMoves      = errors.New("no moves to undo")
)

// Game represents the Gomoku game engine state.
type Game struct {
	mu          sync.RWMutex
	board       [BoardSize][BoardSize]Stone
	currentTurn Stone
	status      GameStatus
	winner      Stone
	history     []Move
	lastMove    *Position
}

// NewGame initializes a new Gomoku game with Black starting first.
func NewGame() *Game {
	return &Game{
		currentTurn: Black,
		status:      Active,
		history:     make([]Move, 0),
	}
}

// ParseCoordinates parses alphanumeric board coordinates (e.g. "H8", "a1", "O15") into a 0-indexed Position.
func ParseCoordinates(coords string) (Position, error) {
	coords = strings.TrimSpace(coords)
	matches := coordRegex.FindStringSubmatch(coords)
	if len(matches) != 3 {
		return Position{}, fmt.Errorf("invalid coordinate format %q: must be A1-O15", coords)
	}

	colLetter := strings.ToUpper(matches[1])
	col := int(colLetter[0] - 'A')

	rowNum, err := strconv.Atoi(matches[2])
	if err != nil {
		return Position{}, fmt.Errorf("invalid row number %q: %w", matches[2], err)
	}
	row := rowNum - 1 // convert 1-index to 0-index

	return Position{Row: row, Col: col}, nil
}

// PlayMoveIndices places a stone at the 0-indexed row and column coordinates.
func (g *Game) PlayMoveIndices(row, col int) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.status != Active {
		return ErrGameOver
	}

	if row < 0 || row >= BoardSize || col < 0 || col >= BoardSize {
		return ErrOutOfBounds
	}

	if g.board[row][col] != Empty {
		return ErrCellOccupied
	}

	player := g.currentTurn
	pos := Position{Row: row, Col: col}
	g.board[row][col] = player
	g.lastMove = &pos

	// Add to history
	g.history = append(g.history, Move{Player: player, Pos: pos})

	// Check win condition
	if g.hasWonAt(pos, player) {
		g.status = Won
		g.winner = player
		return nil
	}

	// Check draw condition
	if len(g.history) == BoardSize*BoardSize {
		g.status = Draw
		return nil
	}

	// Toggle turn
	if g.currentTurn == Black {
		g.currentTurn = White
	} else {
		g.currentTurn = Black
	}

	return nil
}

// PlayMove parses coordinates and places a stone.
func (g *Game) PlayMove(coords string) (Position, error) {
	pos, err := ParseCoordinates(coords)
	if err != nil {
		return Position{}, err
	}
	err = g.PlayMoveIndices(pos.Row, pos.Col)
	if err != nil {
		return Position{}, err
	}
	return pos, nil
}

// Undo rolls back the last move made, reverting turn and state.
func (g *Game) Undo() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.history) == 0 {
		return ErrNoMoves
	}

	lastMoveIndex := len(g.history) - 1
	lastMove := g.history[lastMoveIndex]

	// Revert the board cell to Empty
	g.board[lastMove.Pos.Row][lastMove.Pos.Col] = Empty
	g.history = g.history[:lastMoveIndex]

	// Reset game status
	g.status = Active
	g.winner = Empty

	// Update lastMove reference
	if len(g.history) > 0 {
		prevMove := g.history[len(g.history)-1].Pos
		g.lastMove = &prevMove
	} else {
		g.lastMove = nil
	}

	// Revert turn to the player who made the last move
	g.currentTurn = lastMove.Player

	return nil
}

// hasWonAt checks if the placement at pos results in a win (exactly 5 consecutive stones of same color).
// Implements an optimized scanning algorithm along 4 directions (horizontal, vertical, diagonal-down, diagonal-up).
func (g *Game) hasWonAt(pos Position, color Stone) bool {
	directions := []struct{ dr, dc int }{
		{0, 1},  // Horizontal
		{1, 0},  // Vertical
		{1, 1},  // Diagonal down-right
		{1, -1}, // Diagonal down-left
	}

	for _, d := range directions {
		count := 1

		// Positive direction
		r, c := pos.Row+d.dr, pos.Col+d.dc
		for r >= 0 && r < BoardSize && c >= 0 && c < BoardSize && g.board[r][c] == color {
			count++
			r += d.dr
			c += d.dc
		}

		// Negative direction
		r, c = pos.Row-d.dr, pos.Col-d.dc
		for r >= 0 && r < BoardSize && c >= 0 && c < BoardSize && g.board[r][c] == color {
			count++
			r -= d.dr
			c -= d.dc
		}

		// Check for exactly five consecutive stones
		if count == 5 {
			return true
		}
	}
	return false
}

// BoardString renders the 15x15 board state to a beautiful alphanumeric CLI grid.
func (g *Game) BoardString() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var sb strings.Builder

	// Render column headers
	sb.WriteString("   ")
	for c := 0; c < BoardSize; c++ {
		sb.WriteByte('A' + byte(c))
		if c < BoardSize-1 {
			sb.WriteByte(' ')
		}
	}
	sb.WriteByte('\n')

	// Render rows
	for r := 0; r < BoardSize; r++ {
		sb.WriteString(fmt.Sprintf("%2d ", r+1))
		for c := 0; c < BoardSize; c++ {
			stone := g.board[r][c]
			switch stone {
			case Empty:
				sb.WriteString("+")
			case Black:
				sb.WriteString("●")
			case White:
				sb.WriteString("○")
			}
			if c < BoardSize-1 {
				sb.WriteByte(' ')
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// GetBoard returns a copy of the current board grid.
func (g *Game) GetBoard() [BoardSize][BoardSize]Stone {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.board
}

// GetStatus returns the current game status.
func (g *Game) GetStatus() GameStatus {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.status
}

// GetWinner returns the winning player (or Empty if active/draw).
func (g *Game) GetWinner() Stone {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.winner
}

// GetCurrentTurn returns the stone type whose turn it is.
func (g *Game) GetCurrentTurn() Stone {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.currentTurn
}

// GetHistory returns a copy of the move history.
func (g *Game) GetHistory() []Move {
	g.mu.RLock()
	defer g.mu.RUnlock()
	historyCopy := make([]Move, len(g.history))
	copy(historyCopy, g.history)
	return historyCopy
}

// GetLastMove returns a copy of the last placed position (nil if none).
func (g *Game) GetLastMove() *Position {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.lastMove == nil {
		return nil
	}
	posCopy := *g.lastMove
	return &posCopy
}

// InteractivePlay starts a command-line interface Gomoku game loop.
// It reads input from standard in, writes output to standard out, and gracefully handles interrupt signals.
func (g *Game) InteractivePlay(in io.Reader, out io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	scanner := bufio.NewScanner(in)
	doneChan := make(chan struct{})

	go func() {
		defer close(doneChan)
		for {
			g.mu.RLock()
			status := g.status
			turn := g.currentTurn
			winner := g.winner
			lastPos := g.lastMove
			g.mu.RUnlock()

			// Print board
			fmt.Fprint(out, "\n")
			fmt.Fprint(out, g.BoardString())
			fmt.Fprint(out, "\n")

			if status == Won {
				winnerStr := "Black (●)"
				if winner == White {
					winnerStr = "White (○)"
				}
				fmt.Fprintf(out, "🎉 Game Over! %s wins the game!\n", winnerStr)
				return
			}

			if status == Draw {
				fmt.Fprintln(out, "🤝 Game Over! It's a draw.")
				return
			}

			turnStr := "Black (●)"
			if turn == White {
				turnStr = "White (○)"
			}

			if lastPos != nil {
				lastColLetter := 'A' + byte(lastPos.Col)
				fmt.Fprintf(out, "Last move: %c%d\n", lastColLetter, lastPos.Row+1)
			}
			fmt.Fprintf(out, "Current turn: %s\n", turnStr)
			fmt.Fprint(out, "Enter move (e.g. H8, 'undo', or 'quit'): ")

			if !scanner.Scan() {
				return
			}

			input := strings.TrimSpace(scanner.Text())
			if strings.EqualFold(input, "quit") || strings.EqualFold(input, "exit") {
				fmt.Fprintln(out, "Adios! Thanks for playing.")
				return
			}

			if strings.EqualFold(input, "undo") {
				if err := g.Undo(); err != nil {
					fmt.Fprintf(out, "⚠️ Undo failed: %s\n", err)
				} else {
					fmt.Fprintln(out, "🔄 Last move undone.")
				}
				continue
			}

			_, err := g.PlayMove(input)
			if err != nil {
				fmt.Fprintf(out, "❌ Invalid move: %s\n", err)
			}
		}
	}()

	select {
	case <-ctx.Done():
		fmt.Fprintln(out, "\n👋 Game interrupted. Gracefully shutting down.")
		return nil
	case <-doneChan:
		return nil
	}
}
