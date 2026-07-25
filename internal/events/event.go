// Package events provides an append-only JSONL event stream for observing a
// harness run while it is in flight.
//
// The pipeline's stdout is prose for humans and is written directly by the loop.
// This package is the parallel machine-readable spine: every semantic transition
// is appended as one JSON object per line to workspace/events.jsonl. Monitoring
// UIs attach as separate processes and tail that file, so the run itself stays
// headless and never renders.
//
// The whole package is stdlib-only by design.
package events

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Kind identifies the semantic type of an Event. The payload struct that
// accompanies each Kind is named in its comment.
type Kind string

const (
	KindRunStarted   Kind = "run_started"    // RunStarted
	KindStageChanged Kind = "stage_changed"  // StageChanged
	KindAgentStarted Kind = "agent_started"  // AgentStarted
	KindAgentOutput  Kind = "agent_output"   // AgentOutput
	KindAgentDone    Kind = "agent_done"     // AgentDone
	KindQAResult     Kind = "qa_result"      // QAResult
	KindRetry        Kind = "retry"          // Retry
	KindDelegation   Kind = "delegation"     // Delegation
	KindFileWritten  Kind = "file_written"   // FileWritten
	KindFileRejected Kind = "file_rejected"  // FileRejected
	KindHITLPrompt   Kind = "hitl_prompt"    // HITLPrompt
	KindHITLResolved Kind = "hitl_resolved"  // HITLResolved
	KindRunFinished  Kind = "run_finished"   // RunFinished
)

// Agent roles, used as the Role field on agent events.
const (
	RoleBA     = "ba"
	RoleDev    = "dev"
	RoleDevOps = "devops"
)

// Event is one line of the stream.
//
// Task carries the sub-task (feature) name and is what lets a consumer render
// one lane per module during a parallel epic run, where many goroutines emit
// concurrently. It is empty for run-scoped events.
type Event struct {
	Seq     int             `json:"seq"`     // monotonic from 1; lets a reader detect gaps
	RunID   string          `json:"run_id"`  // distinguishes runs sharing a file
	TS      time.Time       `json:"ts"`      // UTC
	Kind    Kind            `json:"kind"`
	Task    string          `json:"task,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Decode unmarshals the event payload into v.
func (e Event) Decode(v any) error {
	if len(e.Payload) == 0 {
		return fmt.Errorf("event %d (%s) has no payload", e.Seq, e.Kind)
	}
	return json.Unmarshal(e.Payload, v)
}

// ── Payloads ────────────────────────────────────────────────────────────────

// RunStarted is emitted once, before any agent runs.
type RunStarted struct {
	Mode       string            `json:"mode"`        // "task" | "epic" | "resume"
	Task       string            `json:"task,omitempty"`
	Epic       string            `json:"epic,omitempty"`
	ProjectDir string            `json:"project_dir"`
	Parallel   bool              `json:"parallel"`
	MaxRetries int               `json:"max_retries"`
	Agents     map[string]string `json:"agents"` // role -> "agent (model)"
}

// StageChanged mirrors every pipeline state transition.
type StageChanged struct {
	Stage      string `json:"stage"`
	Retry      int    `json:"retry"` // 0-based, as held by the loop
	MaxRetries int    `json:"max_retries"`
}

// AgentStarted precedes every agent CLI invocation.
type AgentStarted struct {
	Role  string `json:"role"`
	Agent string `json:"agent"`
	Model string `json:"model,omitempty"`
}

// AgentOutput is one line of an agent's stdout, mirrored as it streams.
// Long lines are split across several events; see maxChunk in bus.go.
type AgentOutput struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// AgentDone reports the outcome and cost of a single agent invocation.
// Consumers accumulate PromptTokens/EvalTokens across events to track spend
// live; RunFinished carries the authoritative totals.
type AgentDone struct {
	Role         string  `json:"role"`
	PromptTokens int     `json:"prompt_tokens"`
	EvalTokens   int     `json:"eval_tokens"`
	DurationSecs float64 `json:"duration_seconds"`
	Error        string  `json:"error,omitempty"`
}

// QAResult reports a combined security-audit + test-suite verdict.
type QAResult struct {
	Passed     bool   `json:"passed"`
	Target     string `json:"target"`
	Attempt    int    `json:"attempt"` // 1-based, as shown to the user
	MaxRetries int    `json:"max_retries"`
	AuditError string `json:"audit_error,omitempty"`
	TestError  string `json:"test_error,omitempty"`
}

// Retry marks a self-healing attempt after a failed QA pass.
type Retry struct {
	Attempt    int `json:"attempt"` // the attempt that just failed, 1-based
	MaxRetries int `json:"max_retries"`
}

// Delegation marks the BA agent being asked to rewrite failing requirements.
type Delegation struct {
	Cycle    int `json:"cycle"` // 0-based
	MaxCycle int `json:"max_cycle"`
}

// FileWritten records code the Dev agent produced. Action is "write" for a full
// file and "patch" for an applied SEARCH/REPLACE block.
type FileWritten struct {
	Path   string `json:"path"`
	Action string `json:"action"`
}

// FileRejected records a generated path that escaped the target folder and was
// refused. This is a security boundary: the QA audit only scans the target
// folder, so a file written outside it would bypass the scan entirely.
type FileRejected struct {
	Path   string `json:"path"`
	Target string `json:"target"`
	Reason string `json:"reason"`
}

// HITLPrompt is emitted when the loop blocks on the human approval gate.
//
// A monitor can display this gate but cannot answer it: stdin belongs to the
// run process, and the monitor is a separate process. The operator still
// answers in the run's terminal.
type HITLPrompt struct {
	Target         string `json:"target,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// HITLResolved reports how the gate closed. Auto is true when the timeout
// expired and the loop self-approved.
type HITLResolved struct {
	Approved bool `json:"approved"`
	Auto     bool `json:"auto"`
}

// RunFinished is emitted last, including on the abort paths. Reason is set when
// the run ended for a specific cause (QA exhaustion, HITL disapproval).
//
// FailedTasks names the modules that failed in a parallel epic, which completes
// rather than aborting on failure. Success is false whenever it is non-empty.
type RunFinished struct {
	Success            bool     `json:"success"`
	Reason             string   `json:"reason,omitempty"`
	FailedTasks        []string `json:"failed_tasks,omitempty"`
	DurationSeconds    float64  `json:"duration_seconds"`
	TotalPromptTokens  int      `json:"total_prompt_tokens"`
	TotalEvalTokens    int      `json:"total_eval_tokens"`
	TotalRetriesUsed   int      `json:"total_retries_used"`
	CodeHealingSuccess bool     `json:"code_healing_success"`
	LinesOfCode        int      `json:"lines_of_code_generated"`
}

// ── Run identity ────────────────────────────────────────────────────────────

// NewRunID builds a stable, filesystem-safe run identifier from a human label
// and a start time, e.g. "add_login-20260717T101533Z".
//
// This replaces the previously hardcoded workflow TaskID, which labelled every
// run "cti_modular_self_healing" and made runs indistinguishable.
func NewRunID(label string, start time.Time) string {
	s := Slug(label)
	if s == "" {
		s = "run"
	}
	return fmt.Sprintf("%s-%s", s, start.UTC().Format("20060102T150405Z"))
}

// Slug reduces arbitrary text to lowercase alphanumerics joined by underscores,
// truncated to 32 characters.
func Slug(s string) string {
	var b strings.Builder
	lastUnderscore := true // avoids a leading underscore
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastUnderscore = false
		case !lastUnderscore && b.Len() < 32:
			b.WriteRune('_')
			lastUnderscore = true
		}
		if b.Len() >= 32 {
			break
		}
	}
	return strings.Trim(b.String(), "_")
}
