package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dothanhlam/harness-engineering/internal/events"
)

// ev builds an event with a marshalled payload for renderer tests.
func ev(kind events.Kind, task string, payload any) events.Event {
	raw, _ := json.Marshal(payload)
	return events.Event{
		Seq:     1,
		RunID:   "r",
		TS:      time.Date(2026, 7, 18, 14, 32, 1, 0, time.UTC),
		Kind:    kind,
		Task:    task,
		Payload: raw,
	}
}

func TestFormatEvent_Kinds(t *testing.T) {
	cases := []struct {
		name    string
		event   events.Event
		show    bool
		want    bool     // ok
		substrs []string // all must appear when ok
	}{
		{
			name:    "run_started",
			event:   ev(events.KindRunStarted, "", events.RunStarted{Mode: "task", Task: "demo", Agents: map[string]string{"dev": "llama_cpp (g)"}}),
			want:    true,
			substrs: []string{"TASK", "demo", "dev=llama_cpp (g)"},
		},
		{
			name:    "stage_changed shows attempt when retries>1",
			event:   ev(events.KindStageChanged, "demo", events.StageChanged{Stage: "QA_TESTING", Retry: 1, MaxRetries: 3}),
			want:    true,
			substrs: []string{"QA_TESTING", "attempt 2/3", "[demo]"},
		},
		{
			name:    "qa passed",
			event:   ev(events.KindQAResult, "demo", events.QAResult{Passed: true, Target: "workspace/demo"}),
			want:    true,
			substrs: []string{"QA passed", "workspace/demo"},
		},
		{
			name:    "qa failed collapses multiline error",
			event:   ev(events.KindQAResult, "demo", events.QAResult{Passed: false, Attempt: 1, MaxRetries: 3, TestError: "\n# pkg\nsyntax error: bad\nFAIL"}),
			want:    true,
			substrs: []string{"QA failed", "attempt 1/3", "syntax error: bad"},
		},
		{
			name:    "agent_done success",
			event:   ev(events.KindAgentDone, "demo", events.AgentDone{Role: "dev", PromptTokens: 12, EvalTokens: 5, DurationSecs: 1.2}),
			want:    true,
			substrs: []string{"dev done", "12→5 tok", "1.2s"},
		},
		{
			name:    "agent_done error",
			event:   ev(events.KindAgentDone, "demo", events.AgentDone{Role: "dev", Error: "boom\nsecond line"}),
			want:    true,
			substrs: []string{"dev failed", "boom"},
		},
		{
			name:    "file_rejected",
			event:   ev(events.KindFileRejected, "demo", events.FileRejected{Path: "workspace/other/x.go", Target: "workspace/demo", Reason: "path escapes the target folder"}),
			want:    true,
			substrs: []string{"rejected workspace/other/x.go", "path escapes"},
		},
		{
			name:    "hitl_prompt",
			event:   ev(events.KindHITLPrompt, "demo", events.HITLPrompt{TimeoutSeconds: 30}),
			want:    true,
			substrs: []string{"awaiting approval", "auto-yes in 30s"},
		},
		{
			name:    "run_finished success",
			event:   ev(events.KindRunFinished, "", events.RunFinished{Success: true, DurationSeconds: 1.1, LinesOfCode: 10}),
			want:    true,
			substrs: []string{"success", "10 LOC"},
		},
		{
			name:    "run_finished failure carries reason",
			event:   ev(events.KindRunFinished, "", events.RunFinished{Success: false, Reason: "rejected at HITL gate"}),
			want:    true,
			substrs: []string{"FAILED", "rejected at HITL gate"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := FormatEvent(tc.event, false, tc.show)
			if ok != tc.want {
				t.Fatalf("ok = %v, want %v (line=%q)", ok, tc.want, got)
			}
			if !ok {
				return
			}
			if strings.Contains(got, "\n") {
				t.Errorf("line contains a newline: %q", got)
			}
			for _, s := range tc.substrs {
				if !strings.Contains(got, s) {
					t.Errorf("line %q missing %q", got, s)
				}
			}
		})
	}
}

// TestFormatEvent_OutputGate: agent output is shown only when requested.
func TestFormatEvent_OutputGate(t *testing.T) {
	e := ev(events.KindAgentOutput, "demo", events.AgentOutput{Role: "dev", Text: "wrote main.go"})

	if _, ok := FormatEvent(e, false, false); ok {
		t.Error("agent output shown when showOutput=false")
	}
	got, ok := FormatEvent(e, false, true)
	if !ok || !strings.Contains(got, "wrote main.go") {
		t.Errorf("agent output not shown when requested: ok=%v line=%q", ok, got)
	}
}

// TestFormatEvent_UnknownKind is skipped rather than rendered blank, so a future
// event kind never prints a bare timestamp with no content.
func TestFormatEvent_UnknownKind(t *testing.T) {
	e := ev("some_future_kind", "demo", map[string]string{"x": "y"})
	if _, ok := FormatEvent(e, false, true); ok {
		t.Error("unknown kind should be skipped")
	}
}

// TestFormatEvent_ColorToggle: ANSI codes appear only when color is on.
func TestFormatEvent_ColorToggle(t *testing.T) {
	e := ev(events.KindQAResult, "demo", events.QAResult{Passed: true, Target: "t"})

	plain, _ := FormatEvent(e, false, false)
	if strings.Contains(plain, "\033[") {
		t.Errorf("plain output contains ANSI: %q", plain)
	}
	colored, _ := FormatEvent(e, true, false)
	if !strings.Contains(colored, "\033[") {
		t.Errorf("colored output has no ANSI: %q", colored)
	}
}

// syncBuffer is a bytes.Buffer safe for the concurrent writes a live Follow
// makes from its poll goroutine while the test reads.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestFollow_StopsOnLiveRunFinished drives the real loop: history is replayed,
// then a run finishing LIVE (after attach) makes Follow return on its own.
func TestFollow_StopsOnLiveRunFinished(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	bus, err := events.Open(path, "run")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer bus.Close()

	// Pre-existing history (a run in progress), present before Follow attaches.
	bus.Emit(events.KindRunStarted, "", events.RunStarted{Mode: "task", Task: "demo"})
	bus.Emit(events.KindStageChanged, "demo", events.StageChanged{Stage: "DEV_CODING", MaxRetries: 3})

	buf := &syncBuffer{}
	off := false
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- Follow(ctx, path, buf, Options{Color: &off}) }()

	// The run finishes live, after Follow has attached and replayed history.
	waitFor(t, func() bool { return strings.Contains(buf.String(), "DEV_CODING") })
	bus.Emit(events.KindRunFinished, "", events.RunFinished{Success: true, LinesOfCode: 7})
	// Anything after run_finished must not be rendered.
	bus.Emit(events.KindStageChanged, "demo", events.StageChanged{Stage: "SHOULD_NOT_APPEAR"})

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Follow: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("Follow did not stop on a live run_finished")
	}

	out := buf.String()
	for _, want := range []string{"TASK run", "DEV_CODING", "run finished", "7 LOC"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
	if strings.Contains(out, "SHOULD_NOT_APPEAR") {
		t.Errorf("Follow rendered an event after run_finished:\n%s", out)
	}
}

// TestFollow_DoesNotExitOnStaleRun is the regression guard for the bug the
// review caught: a log already ending in run_finished (a prior run that has not
// yet been truncated) must be rendered but must NOT make live follow exit, or a
// monitor started before the next run replays the stale run and quits.
func TestFollow_DoesNotExitOnStaleRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	bus, err := events.Open(path, "old-run")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	bus.Emit(events.KindRunStarted, "", events.RunStarted{Mode: "task", Task: "prev"})
	bus.Emit(events.KindRunFinished, "", events.RunFinished{Success: true})
	bus.Close()

	buf := &syncBuffer{}
	off := false
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- Follow(ctx, path, buf, Options{Color: &off}) }()

	select {
	case <-done:
		t.Fatal("Follow exited on a stale finished run; it should wait for the next run")
	case <-ctx.Done():
		// Correct: it kept waiting and we cancelled it.
	}
	if !strings.Contains(buf.String(), "run finished") {
		t.Errorf("stale history was not replayed: %q", buf.String())
	}
}

// waitFor polls cond up to a short deadline, failing the test if it never holds.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := 2 * time.Second
	step := 10 * time.Millisecond
	for waited := time.Duration(0); waited < deadline; waited += step {
		if cond() {
			return
		}
		time.Sleep(step)
	}
	t.Fatal("condition not met within deadline")
}

// TestFollow_Once replays and returns even without a run_finished.
func TestFollow_Once(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	bus, err := events.Open(path, "run")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	bus.Emit(events.KindStageChanged, "demo", events.StageChanged{Stage: "DEV_CODING", MaxRetries: 3})
	bus.Close()

	var buf bytes.Buffer
	off := false
	if err := Follow(context.Background(), path, &buf, Options{Once: true, Color: &off}); err != nil {
		t.Fatalf("Follow(once): %v", err)
	}
	if !strings.Contains(buf.String(), "DEV_CODING") {
		t.Errorf("once did not render the pending log: %q", buf.String())
	}
}
