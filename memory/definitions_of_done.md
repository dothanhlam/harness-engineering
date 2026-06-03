# Definition of Done: Flappy Bird (Workspace Implementation)

## 1. Core Game Mechanics (Functional)
- [ ] **Gravity Engine:** Implement a constant downward acceleration (vector-based) applied to the bird entity.
- [ ] **Jump Physics:** Implement an instantaneous upward velocity impulse triggered by user input (Space/Click/Touch).
- [ ] **Pipe Generation:** Implement a procedural spawning system for pipes with randomized vertical gaps and fixed horizontal intervals.
- [ ] **Collision Matrix:**
    - [ ] AABB (Axis-Aligned Bounding Box) collision detection between bird and pipes.
    - [ ] Floor collision detection (terminal game state).
    - [ ] Ceiling collision detection (optional: bounce vs. kill).
- [ ] **Scoring System:** Increment score counter exactly when the bird's horizontal trailing edge clears the pipe's horizontal trailing edge.
- [ ] **State Machine:** Implement distinct game states: `START_SCREEN`, `PLAYING`, `GAME_OVER`.

## 2. Technical Architecture
- [ ] **Project Structure:** Source code must reside in `workspace/flappybird/` with clear separation between logic (engine) and rendering (view).
- [ ] **Render Loop:** Implement a requestAnimationFrame-based loop (or equivalent) targeting a stable 60 FPS.
- [ ] **Delta Time Integration:** All physics calculations (movement, gravity) must be normalized using delta time (`dt`) to ensure frame-rate independence.
- [ ] **Asset Management:**
    - [ ] Modular loading of sprites/textures.
    - [ ] Audio manager for jump, score, and collision SFX.
- [ ] **Clean Code Standards:**
    - [ ] No hardcoded magic numbers (extract to `config.js` or `constants.go`).
    - [ ] Documented functions and class structures.

## 3. User Interface & Experience (UI/UX)
- [ ] **Visual Parallax:** (Optional but recommended) Multi-layered background for depth perception.
- [ ] **HUD:** Real-time score display with high contrast and legible typography.
- [ ] **Game Over Overlay:** Display final score and a "high score" persistent record (local storage or session).
- [ ] **Input Latency:** Zero-perceivable lag between input event and physics impulse.
- [ ] **Responsive Design:** Canvas/Container must scale or center correctly within a standard browser window or terminal size.

## 4. Stability & Performance
- [ ] **Memory Management:** Ensure pipes are properly garbage collected/pooled once they exit the left viewport boundary.
- [ ] **Error Handling:** Graceful handling of missing assets (images/sounds) without crashing the engine.
- [ ] **Event Cleanup:** Ensure all event listeners are cleared if the game is restarted or destroyed.

## 5. Deployment & Runnability
- [ ] **Zero-Config Execution:** The game must be runnable via a single command (e.g., `open index.html` or `go run main.go`).
- [ ] **README.md:** Include:
    - [ ] Directives on how to start the game.
    - [ ] Control mapping.
    - [ ] Technical stack overview.
- [ ] **Validation:** Smoke test passed—one full cycle from Start -> Play -> Collision -> Restart.