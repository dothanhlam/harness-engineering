package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dothanhlam/harness-app/internal/telemetry"
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

// WorkflowState is the persisted pipeline state written to workspace/state.json.
type WorkflowState struct {
	TaskID       string    `json:"task_id"`
	CurrentStage Stage     `json:"current_stage"`
	RetryCount   int       `json:"retry_count"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UpdateState writes the current pipeline stage to workspace/state.json and records it in telemetry.
func UpdateState(stage Stage, retry int, tracker *telemetry.Tracker) {
	tracker.AddStage(string(stage))
	state := WorkflowState{
		TaskID:       "cti_modular_self_healing",
		CurrentStage: stage,
		RetryCount:   retry,
		UpdatedAt:    time.Now(),
	}
	file, _ := json.MarshalIndent(state, "", "  ")
	_ = os.WriteFile("workspace/state.json", file, 0644)
	fmt.Printf("\n🔄 [HARNESS STATE] -> %s (Attempt: %d/%d)\n", stage, retry+1, MaxRetries)
}
