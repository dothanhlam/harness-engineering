package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Telemetry holds pipeline execution metrics.
type Telemetry struct {
	TotalDurationSeconds float64  `json:"total_duration_seconds"`
	StagesExecuted       []string `json:"stages_executed"`
	TotalRetriesUsed     int      `json:"total_retries_used"`
	CodeHealingSuccess   bool     `json:"code_healing_success"`
	LinesOfCodeGenerated int      `json:"lines_of_code_generated"`
	TotalPromptTokens    int      `json:"total_prompt_tokens"`
	TotalEvalTokens      int      `json:"total_eval_tokens"`
	Timestamp            string   `json:"timestamp"`
}

// Tracker provides goroutine-safe telemetry collection via sync.Mutex.
// Replaces the global var pipelineTelemetry with an explicit, passable instance.
type Tracker struct {
	mu    sync.Mutex
	data  Telemetry
	start time.Time
}

// NewTracker creates a new Tracker anchored to the given start time.
func NewTracker(start time.Time) *Tracker {
	return &Tracker{start: start}
}

// IncrRetries safely increments the retry counter.
func (t *Tracker) IncrRetries() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data.TotalRetriesUsed++
}

// MarkHealing sets the code healing success flag.
func (t *Tracker) MarkHealing() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data.CodeHealingSuccess = true
}

// AddTokens accumulates token usage into the tracker.
func (t *Tracker) AddTokens(prompt, eval int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data.TotalPromptTokens += prompt
	t.data.TotalEvalTokens += eval
}

// AddStage records a pipeline stage execution with timestamp.
func (t *Tracker) AddStage(stageName string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data.StagesExecuted = append(t.data.StagesExecuted,
		fmt.Sprintf("%s (%s)", stageName, time.Now().Format(time.RFC3339)))
}

// Finalize computes final metrics and writes telemetry JSON to the specified path.
func (t *Tracker) Finalize(outputPath string, linesGenerated int) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data.TotalDurationSeconds = time.Since(t.start).Seconds()
	t.data.LinesOfCodeGenerated = linesGenerated
	t.data.Timestamp = time.Now().Format(time.RFC3339)

	bytes, err := json.MarshalIndent(t.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, bytes, 0644)
}
