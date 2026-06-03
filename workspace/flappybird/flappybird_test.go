package flappybird

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// mockAudio collects audio events triggered during game actions.
type mockAudio struct {
	mu     sync.Mutex
	events []AudioEvent
}

func (m *mockAudio) callback(event AudioEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *mockAudio) getEvents() []AudioEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]AudioEvent, len(m.events))
	copy(copied, m.events)
	return copied
}

func (m *mockAudio) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = make([]AudioEvent, 0)
}

// TestGameInitialization tests that the game is initialized correctly.
func TestGameInitialization(t *testing.T) {
	g := NewGame(Config{Seed: 42})
	if g.GetState() != StateStartScreen {
		t.Errorf("expected state to be StateStartScreen, got %v", g.GetState())
	}
	if g.GetScore() != 0 {
		t.Errorf("expected score to be 0, got %d", g.GetScore())
	}
	bird := g.GetBird()
	if bird.Y != BirdStartY {
		t.Errorf("expected bird Y to be %f, got %f", BirdStartY, bird.Y)
	}
}

// TestFSMTransitions verifies Start -> Playing -> Game Over -> Start transitions.
func TestFSMTransitions(t *testing.T) {
	audio := &mockAudio{}
	g := NewGame(Config{
		AudioCallback: audio.callback,
		Seed:          12345,
	})

	// Start Screen
	if g.GetState() != StateStartScreen {
		t.Fatalf("expected StartScreen, got %v", g.GetState())
	}

	// Flap transitions to Playing
	g.Flap()
	if g.GetState() != StatePlaying {
		t.Errorf("expected Playing after flap, got %v", g.GetState())
	}
	events := audio.getEvents()
	if len(events) != 1 || events[0] != AudioFlap {
		t.Errorf("expected flap audio, got %v", events)
	}
	audio.reset()

	// Apply gravity/update to simulate falling below ground
	// Loop Update until GameOver is hit
	for i := 0; i < 100; i++ {
		g.Update(0.1)
		if g.GetState() == StateGameOver {
			break
		}
	}

	if g.GetState() != StateGameOver {
		t.Errorf("expected GameOver after falling, got %v", g.GetState())
	}

	events = audio.getEvents()
	hasHit := false
	hasGameOver := false
	for _, ev := range events {
		if ev == AudioHit {
			hasHit = true
		}
		if ev == AudioGameOver {
			hasGameOver = true
		}
	}
	if !hasHit || !hasGameOver {
		t.Errorf("expected Hit and GameOver audio events, got %v", events)
	}

	// Restart goes back to Start Screen
	g.Restart()
	if g.GetState() != StateStartScreen {
		t.Errorf("expected StartScreen after restart, got %v", g.GetState())
	}
}

// TestPhysicsMovement verifies constant horizontal movement simulator (via pipe scrolling)
// and gravity vertical acceleration.
func TestPhysicsMovement(t *testing.T) {
	g := NewGame(Config{Seed: 42})
	g.Flap() // Go to Playing state

	birdInitialY := g.GetBird().Y
	// Update by dt = 0.1s
	g.Update(0.1)

	birdAfterUpdate := g.GetBird()
	expectedVel := FlapImpulse + Gravity*0.1
	if mathAbs(birdAfterUpdate.VelocityY-expectedVel) > 1e-9 {
		t.Errorf("expected bird velocity %f, got %f", expectedVel, birdAfterUpdate.VelocityY)
	}

	// Wait, the update Euler or Verlet integration?
	// In flappybird.go:
	//   g.bird.VelocityY += Gravity * dt
	//   g.bird.Y += g.bird.VelocityY * dt
	// This uses Euler integration:
	// VelocityY(t+dt) = VelocityY(t) + Gravity * dt
	// Y(t+dt) = Y(t) + VelocityY(t+dt) * dt
	eulerExpectedY := birdInitialY + (FlapImpulse+Gravity*0.1)*0.1
	if mathAbs(birdAfterUpdate.Y-eulerExpectedY) > 1e-9 {
		t.Errorf("expected bird Y %f, got %f (euler %f)", eulerExpectedY, birdAfterUpdate.Y, eulerExpectedY)
	}
}

// TestRapidInputBuffer verifies that rapid inputs spamming doesn't break physics constraints
// (velocity is directly set to FlapImpulse on each flap rather than adding, so it behaves deterministically).
func TestRapidInputBuffer(t *testing.T) {
	g := NewGame(Config{Seed: 42})
	g.Flap() // Start game

	for i := 0; i < 50; i++ {
		g.Flap()
	}

	bird := g.GetBird()
	if bird.VelocityY != FlapImpulse {
		t.Errorf("expected VelocityY to be exactly %f, got %f", FlapImpulse, bird.VelocityY)
	}
}

// TestCollisionDetection verifies AABB pipe boundary collision.
func TestCollisionDetection(t *testing.T) {
	g := NewGame(Config{Seed: 42})
	g.Flap() // Start playing

	// Let's manually inject a pipe pair that overlaps with the bird's constant X = 100.0
	// The bird hitbox is centered at X=100, Y=240, width=34, height=24.
	// Bounding box: Left=83, Right=117, Top=228, Bottom=252.
	g.mu.Lock()
	g.pipes = []PipePair{
		{
			X:            90.0, // Pipe width is 52. Left=90, Right=142. Overlaps horizontally!
			TopHeight:    230.0, // Top pipe bottom is Y=230. Bird top is Y=228. This is a collision!
			BottomHeight: 50.0,
			Scored:       false,
		},
	}
	g.mu.Unlock()

	g.Update(0.01) // Run update
	if g.GetState() != StateGameOver {
		t.Error("expected collision with top pipe, but game is still active")
	}

	// Reset and check collision with bottom pipe
	g.Reset()
	g.Flap()
	g.mu.Lock()
	g.pipes = []PipePair{
		{
			X:            90.0,  // Overlaps horizontally
			TopHeight:    100.0, // Top pipe is safe
			BottomHeight: GroundY - 245.0, // Bottom pipe top Y = GroundY - BottomHeight = 245.0. Bird bottom is 252.0. Collision!
			Scored:       false,
		},
	}
	g.mu.Unlock()

	g.Update(0.01)
	if g.GetState() != StateGameOver {
		t.Error("expected collision with bottom pipe, but game is still active")
	}
}

// TestScoringLogic verifies score increments once on trailing edge pass.
func TestScoringLogic(t *testing.T) {
	audio := &mockAudio{}
	g := NewGame(Config{
		AudioCallback: audio.callback,
		Seed:          42,
	})
	g.Flap()
	audio.reset()

	// Inject a pipe that is to the right of the bird, but close to passing
	// Bird is at X=100, width=34 (Left=83, Right=117).
	// Let's place a pipe at X = 80. PipeWidth = 52. Left=80, Right=132.
	// Not passed trailing edge yet because Right=132 is greater than bird Left=83 (or trailing edge of bird).
	// Wait, scoring condition: BirdStartX - BirdWidth/2 > pipe.X + PipeWidth
	// BirdStartX - BirdWidth/2 = 100 - 17 = 83.
	// So we need pipe.X + PipeWidth < 83.
	// If pipe.X = 30. pipe.X + PipeWidth = 82 < 83. It should score!
	g.mu.Lock()
	g.pipes = []PipePair{
		{
			X:            50.0,  // X + PipeWidth = 102. Not scored yet.
			TopHeight:    10.0,  // Safe heights
			BottomHeight: 10.0,
			Scored:       false,
		},
	}
	g.mu.Unlock()

	// Update game. Pipes move left by HorizontalSpeed (120 px/s) * dt.
	// We need total dt = 0.2s. Since Update clamps dt at 0.1s, we call it twice.
	g.Update(0.1)
	g.Update(0.1)

	if g.GetScore() != 1 {
		t.Errorf("expected score of 1, got %d", g.GetScore())
	}
	if len(g.GetPipes()) != 1 || !g.GetPipes()[0].Scored {
		t.Error("expected pipe to be marked as Scored")
	}

	events := audio.getEvents()
	if len(events) != 1 || events[0] != AudioScore {
		t.Errorf("expected score audio event, got %v", events)
	}

	// Update again to make sure score does NOT double increment
	g.Update(0.1)
	if g.GetScore() != 1 {
		t.Errorf("expected score to remain 1, got %d", g.GetScore())
	}
}

// TestHighScorePersistence verifies saving and loading to store.
func TestHighScorePersistence(t *testing.T) {
	store := &InMemoryStore{}
	g := NewGame(Config{
		Store: store,
		Seed:  42,
	})
	g.Flap()

	// Trigger a score increment
	g.mu.Lock()
	g.pipes = []PipePair{
		{
			X:            30.0, // Already passed bird X
			TopHeight:    10.0,
			BottomHeight: 10.0,
			Scored:       false,
		},
	}
	g.mu.Unlock()

	g.Update(0.01)

	if g.GetScore() != 1 {
		t.Fatalf("expected score 1, got %d", g.GetScore())
	}

	high, err := store.LoadHighScore()
	if err != nil {
		t.Fatalf("unexpected error loading high score: %v", err)
	}
	if high != 1 {
		t.Errorf("expected high score in store to be 1, got %d", high)
	}

	// Restart game, verify high score is preserved
	g.Reset()
	if g.GetHighScore() != 1 {
		t.Errorf("expected high score to be preserved as 1, got %d", g.GetHighScore())
	}
}

// TestHandler_Index verifies the root HTML template loading and key tags.
func TestHandler_Index(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("Failed to execute GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected HTML Content-Type, got %q", contentType)
	}

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	bodyStr := buf.String()

	keywords := []string{
		"<!DOCTYPE html>",
		"CYBER_FLAP",
		"gameCanvas",
		"startButton",
	}

	for _, kw := range keywords {
		if !strings.Contains(bodyStr, kw) {
			t.Errorf("Expected HTML to contain %q, but it was missing", kw)
		}
	}
}

// TestHandler_CSS verifies style.css is served with correct type.
func TestHandler_CSS(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/style.css")
	if err != nil {
		t.Fatalf("Failed to execute GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/css") {
		t.Errorf("Expected CSS Content-Type, got %q", contentType)
	}
}

// TestHandler_JS verifies app.js is served with correct type.
func TestHandler_JS(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/app.js")
	if err != nil {
		t.Fatalf("Failed to execute GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/javascript") {
		t.Errorf("Expected JS Content-Type, got %q", contentType)
	}
}

// TestHandler_NotFound verifies invalid route returns 404.
func TestHandler_NotFound(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/invalid-route-404")
	if err != nil {
		t.Fatalf("Failed to execute GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func mathAbs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
