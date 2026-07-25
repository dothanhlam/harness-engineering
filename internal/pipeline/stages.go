package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dothanhlam/harness-engineering/internal/events"
	"github.com/dothanhlam/harness-engineering/internal/telemetry"
)

// Stage represents a pipeline execution stage.
type Stage string

const (
	StageDev        Stage = "DEV_CODING"
	StageQA         Stage = "QA_TESTING"
	StageBARefactor Stage = "BA_REFACTOR"
	StageHITL       Stage = "HUMAN_IN_THE_LOOP"
	StageCompact    Stage = "MEMORY_COMPACTION"
	StageDevOps     Stage = "DEVOPS_DELIVER"
	StageDone       Stage = "COMPLETED"
)

// MaxRetries is the maximum number of self-healing attempts per delegation cycle.
const MaxRetries = 3

// hitlTimeout is how long the human-approval gate waits before self-approving.
const hitlTimeout = 30 * time.Second

// devTimeout bounds a single llama.cpp Dev agent invocation. A local model that
// does not fit in memory does not fail — it thrashes, taking the host down with
// it and stalling the run indefinitely. Generous enough for a real generation
// pass on a large feature, short enough that a wedged load surfaces as an error
// the self-healing loop can act on. Only the llama.cpp path is bounded; a CLI
// agent like Claude has its own transport timeouts.
const devTimeout = 10 * time.Minute

// WorkflowState is the persisted pipeline state written to workspace/state.json.
//
// This is a point-in-time snapshot that each transition overwrites, and in epic
// mode every sub-task overwrites the same file. It is kept for backward
// compatibility; the event stream is the ordered, per-task record.
type WorkflowState struct {
	TaskID       string    `json:"task_id"`
	CurrentStage Stage     `json:"current_stage"`
	RetryCount   int       `json:"retry_count"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UpdateState writes the current pipeline stage to workspace/state.json, records
// it in telemetry, and emits it to the event stream. task names the sub-task
// being worked on and may be empty for run-scoped transitions.
func UpdateState(stage Stage, task string, retry int, tracker *telemetry.Tracker, bus *events.Bus) {
	tracker.AddStage(string(stage))

	// TaskID was previously hardcoded, labelling every run identically. Prefer
	// the sub-task, then the run, then the old constant as a last resort.
	taskID := task
	if taskID == "" {
		taskID = bus.RunID()
	}
	if taskID == "" {
		taskID = "harness_run"
	}

	state := WorkflowState{
		TaskID:       taskID,
		CurrentStage: stage,
		RetryCount:   retry,
		UpdatedAt:    time.Now(),
	}
	file, _ := json.MarshalIndent(state, "", "  ")
	_ = os.WriteFile("workspace/state.json", file, 0644)

	emitStage(bus, stage, task, retry)

	fmt.Printf("\n🔄 [HARNESS STATE] -> %s (Attempt: %d/%d)\n", stage, retry+1, MaxRetries)
}

// emitStage records a transition on the event stream only, touching neither
// state.json nor stdout.
//
// The parallel epic path needs this: its sub-task goroutines run concurrently,
// and UpdateState's os.WriteFile to the single workspace/state.json would race
// between them. Bus.Emit is mutex-guarded and safe to call from any goroutine.
func emitStage(bus *events.Bus, stage Stage, task string, retry int) {
	bus.Emit(events.KindStageChanged, task, events.StageChanged{
		Stage:      string(stage),
		Retry:      retry,
		MaxRetries: MaxRetries,
	})
}
