// Package flappybird implements a complete, thread-safe Flappy Bird game engine.
// It includes physics simulation, state management via a Finite State Machine (FSM),
// procedural pipe generation, collision detection (AABB), scoring logic, high score persistence,
// and audio event triggers.
package flappybird

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

// Game constants matching standard Flappy Bird mechanics and dimensions.
const (
	ScreenWidth        = 480.0  // Logical width of the game screen in pixels
	ScreenHeight       = 640.0  // Logical height of the game screen in pixels
	GroundY            = 560.0  // Y-coordinate of the ground boundary
	CeilingY           = 0.0    // Y-coordinate of the ceiling boundary
	BirdWidth          = 34.0   // Width of the bird sprite hitbox
	BirdHeight         = 24.0   // Height of the bird sprite hitbox
	BirdStartX         = 100.0  // Constant X position of the bird
	BirdStartY         = 240.0  // Starting Y position of the bird
	PipeWidth          = 52.0   // Width of the pipe sprite hitbox
	PipeGap            = 120.0  // Vertical gap between upper and lower pipes
	MinPipeHeight      = 50.0   // Minimum height of a pipe
	MaxPipeHeight      = 350.0  // Maximum height of a pipe
	Gravity            = 800.0  // Acceleration downwards in pixels/s^2
	FlapImpulse        = -250.0 // Discrete upward velocity applied on flap in pixels/s
	HorizontalSpeed    = 120.0  // Horizontal speed of the game (pipes moving left) in pixels/s
	PipeSpawnInterval  = 1.5    // Time interval between pipe spawns in seconds
	FloatingAmplitude  = 8.0    // Floating amplitude in start screen
	FloatingSpeed      = 4.0    // Floating speed in start screen
)

// State represents the current phase of the game in the Finite State Machine (FSM).
type State int

const (
	StateStartScreen State = iota // Waiting for user's initial input
	StatePlaying                 // Active gameplay
	StateGameOver                // Game ended due to collision
)

// AudioEvent represents a trigger for sound effects.
type AudioEvent string

const (
	AudioFlap     AudioEvent = "flap"
	AudioScore    AudioEvent = "score"
	AudioHit      AudioEvent = "hit"
	AudioGameOver AudioEvent = "gameover"
)

// Bird represents the player entity.
type Bird struct {
	Y         float64 // Vertical position of the center of the bird
	VelocityY float64 // Vertical velocity of the bird
	Width     float64 // Hitbox width
	Height    float64 // Hitbox height
}

// PipePair represents a vertical obstacle consisting of an upper and lower pipe.
type PipePair struct {
	X            float64 // Horizontal position of the left edge of the pipe pair
	TopHeight    float64 // Height of the top pipe (extending down from Y=0)
	BottomHeight float64 // Height of the bottom pipe (extending up from GroundY)
	Scored       bool    // Whether the player has already scored for passing this pipe pair
}

// HighScoreStore defines the persistence interface for the game's high score.
type HighScoreStore interface {
	SaveHighScore(score int) error
	LoadHighScore() (int, error)
}

// InMemoryStore implements HighScoreStore in memory (no disk I/O, ideal for testing).
type InMemoryStore struct {
	highScore int
	mu        sync.Mutex
}

// SaveHighScore saves the high score to memory.
func (m *InMemoryStore) SaveHighScore(score int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.highScore = score
	return nil
}

// LoadHighScore loads the high score from memory.
func (m *InMemoryStore) LoadHighScore() (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.highScore, nil
}

// Game represents the state and configuration of the Flappy Bird game engine.
type Game struct {
	mu            sync.RWMutex
	bird          Bird
	pipes         []PipePair
	score         int
	highScore     int
	state         State
	spawnTimer    float64
	runningTime   float64
	store         HighScoreStore
	audioCallback func(AudioEvent)
	rng           *rand.Rand
}

// Config provides configuration options when initializing a new Game.
type Config struct {
	Store         HighScoreStore      // Custom persistence store. If nil, InMemoryStore is used.
	AudioCallback func(AudioEvent)    // Callback triggered when SFX events occur.
	Seed          int64               // Seed for the random number generator. If 0, uses current time.
}

// NewGame initializes and returns a new Game engine.
func NewGame(cfg Config) *Game {
	store := cfg.Store
	if store == nil {
		store = &InMemoryStore{}
	}

	seed := cfg.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(seed))

	highScore, _ := store.LoadHighScore()

	g := &Game{
		state:         StateStartScreen,
		store:         store,
		audioCallback: cfg.AudioCallback,
		rng:           rng,
		highScore:     highScore,
	}
	g.Reset()
	return g
}

// Reset resets the game state back to the Start Screen.
func (g *Game) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.bird = Bird{
		Y:      BirdStartY,
		Width:  BirdWidth,
		Height: BirdHeight,
	}
	g.pipes = make([]PipePair, 0)
	g.score = 0
	g.state = StateStartScreen
	g.spawnTimer = 0.0
	g.runningTime = 0.0
}

// GetState returns the current game FSM state.
func (g *Game) GetState() State {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.state
}

// GetBird returns a copy of the bird state.
func (g *Game) GetBird() Bird {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.bird
}

// GetPipes returns a copy of the pipe pairs.
func (g *Game) GetPipes() []PipePair {
	g.mu.RLock()
	defer g.mu.RUnlock()
	pipesCopy := make([]PipePair, len(g.pipes))
	copy(pipesCopy, g.pipes)
	return pipesCopy
}

// GetScore returns the current score.
func (g *Game) GetScore() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.score
}

// GetHighScore returns the high score.
func (g *Game) GetHighScore() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.highScore
}

// Flap triggers the jump mechanics.
// If the game is in StateStartScreen, this transitions to StatePlaying.
// If the game is in StatePlaying, this applies an upward impulse.
// If the game is in StateGameOver, this is a no-op (use Restart).
func (g *Game) Flap() {
	g.mu.Lock()
	defer g.mu.Unlock()

	switch g.state {
	case StateStartScreen:
		g.state = StatePlaying
		g.bird.VelocityY = FlapImpulse
		g.triggerAudio(AudioFlap)
	case StatePlaying:
		g.bird.VelocityY = FlapImpulse
		g.triggerAudio(AudioFlap)
	case StateGameOver:
		// No-op. Player must explicitly call Restart to reset.
	}
}

// Restart transitions the game back to the Start Screen and resets state.
func (g *Game) Restart() {
	g.Reset()
}

// Update advances the game physics and state by delta time (dt) in seconds.
// dt is clamped to prevent wild physics jumps on low frame rates.
func (g *Game) Update(dt float64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Clamp dt to a maximum of 0.1s to avoid physics instability (e.g. passing through obstacles)
	if dt > 0.1 {
		dt = 0.1
	}
	if dt <= 0.0 {
		return
	}

	switch g.state {
	case StateStartScreen:
		g.runningTime += dt
		// Gentle floating physics using sine wave
		g.bird.Y = BirdStartY + FloatingAmplitude*math.Sin(FloatingSpeed*g.runningTime)
		g.bird.VelocityY = 0.0

	case StatePlaying:
		g.runningTime += dt

		// 1. Update Bird Physics
		g.bird.VelocityY += Gravity * dt
		g.bird.Y += g.bird.VelocityY * dt

		// 2. Ceiling and Ground Collisions
		if g.bird.Y-g.bird.Height/2 <= CeilingY {
			g.bird.Y = CeilingY + g.bird.Height/2
			g.handleCollision()
			return
		}
		if g.bird.Y+g.bird.Height/2 >= GroundY {
			g.bird.Y = GroundY - g.bird.Height/2
			g.handleCollision()
			return
		}

		// 3. Update Pipe Obstacles
		g.spawnTimer += dt
		if g.spawnTimer >= PipeSpawnInterval {
			g.spawnTimer -= PipeSpawnInterval
			g.spawnPipePair()
		}

		for i := 0; i < len(g.pipes); i++ {
			g.pipes[i].X -= HorizontalSpeed * dt

			// Check collision with bird
			if g.checkCollision(g.pipes[i]) {
				g.handleCollision()
				return
			}

			// Scoring check: bird passes trailing edge of pipe
			if !g.pipes[i].Scored && BirdStartX-BirdWidth/2 > g.pipes[i].X+PipeWidth {
				g.pipes[i].Scored = true
				g.score++
				g.triggerAudio(AudioScore)

				if g.score > g.highScore {
					g.highScore = g.score
					_ = g.store.SaveHighScore(g.highScore)
				}
			}
		}

		// 4. Cleanup off-screen pipes
		activePipes := make([]PipePair, 0, len(g.pipes))
		for _, pipe := range g.pipes {
			if pipe.X+PipeWidth >= 0 {
				activePipes = append(activePipes, pipe)
			}
		}
		g.pipes = activePipes

	case StateGameOver:
		// No physics updates during Game Over
	}
}

// spawnPipePair creates a new pipe obstacle with randomized gap Y positions.
func (g *Game) spawnPipePair() {
	// The gap height is fixed. Randomize top height.
	// Bottom height fits remaining height to the ground.
	maxTopHeight := GroundY - PipeGap - MinPipeHeight
	if maxTopHeight > MaxPipeHeight {
		maxTopHeight = MaxPipeHeight
	}

	heightRange := maxTopHeight - MinPipeHeight
	topHeight := MinPipeHeight
	if heightRange > 0 {
		topHeight += g.rng.Float64() * heightRange
	}
	bottomHeight := GroundY - topHeight - PipeGap

	g.pipes = append(g.pipes, PipePair{
		X:            ScreenWidth,
		TopHeight:    topHeight,
		BottomHeight: bottomHeight,
		Scored:       false,
	})
}

// checkCollision checks if the bird intersects with a given pipe pair using AABB.
func (g *Game) checkCollision(pipe PipePair) bool {
	birdLeft := BirdStartX - BirdWidth/2
	birdRight := BirdStartX + BirdWidth/2
	birdTop := g.bird.Y - g.bird.Height/2
	birdBottom := g.bird.Y + g.bird.Height/2

	pipeLeft := pipe.X
	pipeRight := pipe.X + PipeWidth

	// Horizontal overlap check
	if birdRight < pipeLeft || birdLeft > pipeRight {
		return false
	}

	// Vertical overlap with Top Pipe (extends from Y=0 to Y=TopHeight)
	if birdTop < pipe.TopHeight {
		return true
	}

	// Vertical overlap with Bottom Pipe (extends from Y=GroundY-BottomHeight to Y=GroundY)
	bottomPipeTopY := GroundY - pipe.BottomHeight
	if birdBottom > bottomPipeTopY {
		return true
	}

	return false
}

// handleCollision transitions the game state to GameOver and triggers relevant sounds.
func (g *Game) handleCollision() {
	g.state = StateGameOver
	g.triggerAudio(AudioHit)
	g.triggerAudio(AudioGameOver)
}

// triggerAudio runs the audio callback if defined.
func (g *Game) triggerAudio(event AudioEvent) {
	if g.audioCallback != nil {
		// Callback in a non-blocking/isolated way or direct call depending on client safety
		g.audioCallback(event)
	}
}

// SetHighScoreStore allows dynamic swapping of the high score store mechanism.
func (g *Game) SetHighScoreStore(store HighScoreStore) error {
	if store == nil {
		return errors.New("store cannot be nil")
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	g.store = store
	highScore, err := store.LoadHighScore()
	if err == nil {
		g.highScore = highScore
	}
	return err
}
