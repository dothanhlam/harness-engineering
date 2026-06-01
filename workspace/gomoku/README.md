# Gomoku (Go-moku) Modular Game Engine

A clean, thread-safe, high-performance modular package implementing a classic **Gomoku** (Five in a Row) game engine and CLI renderer in Go.

---

## 1. Core Architecture

The package contains a strict separation between domain modeling (Game Engine) and User Interface (rendering).

### State Model (`Game`)
The `Game` struct encapsulates the entire game state:
- **Board Grid:** A standard 15x15 matrix (`[15][15]Stone`) initializing with empty cells (`+`).
- **Turn Tracking:** Standard turn management alternating between Black (`●`, Player 1) and White (`○`, Player 2) with Black starting first.
- **Move History:** A slice of executed moves to facilitate dynamic `Undo` rollback.
- **Concurrency Safety:** Thread-safe state access and mutation utilizing an optimized read-write mutual exclusion lock (`sync.RWMutex`).

---

## 2. Technical Features

### Alphanumeric Coordinate Parsing
- Validates grid inputs via a rigid regular expression: `^(?i)([A-O])(1[0-5]|[1-9])$`.
- Sanitizes case-insensitivity and whitespace boundaries seamlessly (e.g. ` h8 ` -> row 7, col 7).

### Optimized Win Check
- Scans four axes (horizontal, vertical, diagonal-up, diagonal-down) originating from the last placed move.
- Completes in deterministic $O(1)$ time complexity (maximum of 36 lookups).
- Detects **exactly five** consecutive stones of the same color to win, preventing overline sequences (6 or more consecutive stones) from triggering a false victory.

### Interactive CLI Loop
- `InteractivePlay(io.Reader, io.Writer)` drives an active CLI loop displaying a beautiful board.
- Supports graceful interrupt handling (Ctrl+C) utilizing `signal.NotifyContext`.
- Emits clear visual feedback showing the board grid, current turn, last move placed, and game outcome status.

---

## 3. How to Build & Run Tests

Verify full compliance and zero compiler issues using standard Go tools from the root folder:

### Run Unit, Integration & Benchmark Tests
```bash
go test -v ./workspace/gomoku/...
```

### Run Performance Benchmarks
```bash
go test -bench=. -benchmem ./workspace/gomoku/...
```
