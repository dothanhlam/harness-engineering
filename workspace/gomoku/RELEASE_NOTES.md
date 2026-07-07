- Gomoku game engine and CLI renderer implemented in Go
- Thread-safe and high-performance
- Compliant with project-specific definitions of done
- 15x15 board size
- Black (●) and White (○) stone types
- Game status: Active, Won, Draw
- ParseCoordinates function for alphanumeric board coordinates parsing
- PlayMoveIndices and PlayMove functions for placing stones and handling input
- Undo function to rollback moves
- hasWonAt function for win condition checking
- BoardString function for rendering the board state
- InteractivePlay for command-line interface game loop
- Unit tests covering parsing, move validation, win conditions, undo, concurrency safety, and integration scenarios
- Benchmark for play move and win checking performance

> EOF by user