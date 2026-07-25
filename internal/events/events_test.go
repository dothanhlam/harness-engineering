package events

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

// collect follows path until it has n events or the deadline passes.
func collect(t *testing.T, path string, n int) []Event {
	t.Helper()

	tl := NewTailer(path)
	tl.Poll = 5 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ch := make(chan Event, n*2)
	go func() { _ = tl.Follow(ctx, func(e Event) { ch <- e }) }()

	var got []Event
	for len(got) < n {
		select {
		case ev := <-ch:
			got = append(got, ev)
		case <-ctx.Done():
			t.Fatalf("timed out after %d/%d events", len(got), n)
		}
	}
	return got
}

// TestBusTailerRoundTrip is the contract every monitor depends on: what the
// pipeline emits is what a separate process reads back, in order.
func TestBusTailerRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	bus, err := Open(path, "test-run-1")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer bus.Close()

	bus.Emit(KindStageChanged, "db_connector", StageChanged{Stage: "DEV_CODING", Retry: 0, MaxRetries: 3})
	bus.Emit(KindQAResult, "db_connector", QAResult{Passed: false, Attempt: 1, TestError: "boom"})

	got := collect(t, path, 2)

	if got[0].Seq != 1 || got[1].Seq != 2 {
		t.Errorf("Seq = %d,%d; want 1,2", got[0].Seq, got[1].Seq)
	}
	if got[0].RunID != "test-run-1" {
		t.Errorf("RunID = %q, want test-run-1", got[0].RunID)
	}
	if got[0].Task != "db_connector" {
		t.Errorf("Task = %q, want db_connector", got[0].Task)
	}
	if got[0].TS.IsZero() {
		t.Error("TS not stamped")
	}

	var stage StageChanged
	if err := got[0].Decode(&stage); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if stage.Stage != "DEV_CODING" || stage.MaxRetries != 3 {
		t.Errorf("StageChanged = %+v", stage)
	}

	var qa QAResult
	if err := got[1].Decode(&qa); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if qa.Passed || qa.TestError != "boom" {
		t.Errorf("QAResult = %+v", qa)
	}
}

// TestNilBusIsNoOp guards the promise that monitoring can never take the
// pipeline down: every entry point must tolerate a nil Bus.
func TestNilBusIsNoOp(t *testing.T) {
	var bus *Bus // nil

	bus.Emit(KindStageChanged, "t", StageChanged{Stage: "DEV_CODING"})
	if got := bus.RunID(); got != "" {
		t.Errorf("RunID = %q, want empty", got)
	}
	if w := bus.AgentWriter("t", RoleDev); w != nil {
		t.Errorf("AgentWriter = %v, want nil", w)
	}
	if err := bus.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	Flush(nil)
}

// TestAgentWriterSplitsLines checks stdout mirroring: one event per line, CR
// stripped, and a trailing partial line surfaced only on Flush.
func TestAgentWriterSplitsLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	bus, err := Open(path, "run")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer bus.Close()

	w := bus.AgentWriter("mod", RoleDev)

	// Split mid-line across writes: the reassembled line must come back whole.
	if n, err := w.Write([]byte("hello wo")); err != nil || n != 8 {
		t.Fatalf("Write = %d, %v", n, err)
	}
	if _, err := w.Write([]byte("rld\r\nsecond\ntrail")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got := collect(t, path, 2)
	var first, second AgentOutput
	_ = got[0].Decode(&first)
	_ = got[1].Decode(&second)

	if first.Text != "hello world" {
		t.Errorf("first = %q, want %q", first.Text, "hello world")
	}
	if first.Role != RoleDev {
		t.Errorf("Role = %q, want %q", first.Role, RoleDev)
	}
	if second.Text != "second" {
		t.Errorf("second = %q, want %q", second.Text, "second")
	}

	// "trail" has no newline, so it stays buffered until flushed.
	Flush(w)
	all := collect(t, path, 3)
	var third AgentOutput
	_ = all[2].Decode(&third)
	if third.Text != "trail" {
		t.Errorf("third = %q, want %q", third.Text, "trail")
	}
}

// TestAgentWriterChunksUnboundedOutput covers a model emitting a huge line, or
// no newline at all: the writer must not buffer without bound, and must never
// report an error (io.MultiWriter would abandon the agent's terminal output).
func TestAgentWriterChunksUnboundedOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	bus, err := Open(path, "run")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer bus.Close()

	w := bus.AgentWriter("mod", RoleDev)
	blob := strings.Repeat("x", maxChunk*2+10) // no newline anywhere
	n, err := w.Write([]byte(blob))
	if err != nil || n != len(blob) {
		t.Fatalf("Write = %d, %v; want %d, nil", n, err, len(blob))
	}

	got := collect(t, path, 2)
	for i, ev := range got {
		var out AgentOutput
		if err := ev.Decode(&out); err != nil {
			t.Fatalf("Decode: %v", err)
		}
		if len(out.Text) > maxChunk {
			t.Errorf("event %d text len = %d, exceeds maxChunk %d", i, len(out.Text), maxChunk)
		}
	}
}

// TestConcurrentEmit mirrors the parallel epic path: many goroutines emitting at
// once must produce unique, gap-free sequence numbers. Run with -race.
func TestConcurrentEmit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	bus, err := Open(path, "run")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer bus.Close()

	const goroutines, per = 8, 25
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < per; i++ {
				bus.Emit(KindFileWritten, fmt.Sprintf("task_%d", g), FileWritten{
					Path:   fmt.Sprintf("workspace/task_%d/f%d.go", g, i),
					Action: "write",
				})
			}
		}(g)
	}
	wg.Wait()

	all, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(all) != goroutines*per {
		t.Fatalf("got %d events, want %d", len(all), goroutines*per)
	}

	seen := make(map[int]bool, len(all))
	for _, ev := range all {
		if seen[ev.Seq] {
			t.Fatalf("duplicate Seq %d", ev.Seq)
		}
		seen[ev.Seq] = true
	}
	for i := 1; i <= goroutines*per; i++ {
		if !seen[i] {
			t.Errorf("missing Seq %d", i)
		}
	}
}

// TestTailerSkipsTornLine covers a run killed mid-write: the reader must hold
// back a partial line rather than parse it, then deliver it once complete.
func TestTailerSkipsTornLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer f.Close()

	tl := NewTailer(path)
	var got []Event
	fn := func(e Event) { got = append(got, e) }

	// A complete line, then a truncated one.
	_, _ = f.WriteString(`{"seq":1,"run_id":"r","kind":"retry"}` + "\n")
	_, _ = f.WriteString(`{"seq":2,"run_id":"r","ki`)
	if err := tl.drain(fn); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(got) != 1 || got[0].Seq != 1 {
		t.Fatalf("got %d events, want only seq 1", len(got))
	}

	// The rest of the line arrives; now it should be delivered exactly once.
	_, _ = f.WriteString(`nd":"retry"}` + "\n")
	if err := tl.drain(fn); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(got) != 2 || got[1].Seq != 2 || got[1].Kind != KindRetry {
		t.Fatalf("got %+v, want seq 2 retry appended", got)
	}
}

// TestTailerDetectsTruncation covers a new run truncating the log while a
// monitor is attached: the reader must rewind rather than wait at a stale
// offset for the file to exceed its old length.
func TestTailerDetectsTruncation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	bus, err := Open(path, "run-1")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	for i := 0; i < 5; i++ {
		bus.Emit(KindRetry, "t", Retry{Attempt: i})
	}
	bus.Close()

	tl := NewTailer(path)
	var got []Event
	if err := tl.drain(func(e Event) { got = append(got, e) }); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("got %d events, want 5", len(got))
	}

	// A second run truncates and writes a single, shorter event.
	bus2, err := Open(path, "run-2")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer bus2.Close()
	bus2.Emit(KindRunStarted, "", RunStarted{Mode: "task"})

	got = nil
	if err := tl.drain(func(e Event) { got = append(got, e) }); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d events after truncation, want 1", len(got))
	}
	if got[0].RunID != "run-2" || got[0].Seq != 1 {
		t.Errorf("got %+v, want run-2 seq 1", got[0])
	}
}

// TestLargePayloadStaysReadable is a regression test: an uncapped `go test -v`
// dump in QAResult.TestError produced a JSONL line past ReadAll's scanner limit,
// and the scanner then failed the ENTIRE read — a monitor showed an empty run at
// exactly the moment the operator needed to see the failure.
func TestLargePayloadStaysReadable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	bus, err := Open(path, "run")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer bus.Close()

	huge := strings.Repeat("FAIL output line\n", 80000) // ~1.4MB
	bus.Emit(KindQAResult, "demo", QAResult{
		Passed:    false,
		TestError: TruncateText(huge, MaxErrorText),
	})
	bus.Emit(KindRunFinished, "", RunFinished{Success: false})

	all, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d events, want 2 — the log became unreadable", len(all))
	}

	var qa QAResult
	if err := all[0].Decode(&qa); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(qa.TestError) > MaxErrorText+128 {
		t.Errorf("TestError len = %d, want <= MaxErrorText plus the truncation note", len(qa.TestError))
	}
	if !strings.HasPrefix(qa.TestError, "FAIL output line") {
		t.Error("truncation dropped the head of the error; the first failure is what a monitor shows")
	}
	if !strings.Contains(qa.TestError, "truncated") {
		t.Error("truncation is silent; the reader cannot tell text was dropped")
	}
}

// TestTruncateTextPreservesRunes guards against slicing a multi-byte character
// in half, which json.Marshal would turn into U+FFFD.
func TestTruncateTextPreservesRunes(t *testing.T) {
	// "世" is 3 bytes; cutting at 10 bytes lands mid-rune.
	s := strings.Repeat("世", 8)
	got := TruncateText(s, 10)
	head := strings.SplitN(got, "\n", 2)[0]
	if !utf8.ValidString(head) {
		t.Errorf("truncated head is not valid UTF-8: %q", head)
	}
	if head != strings.Repeat("世", 3) {
		t.Errorf("head = %q, want 3 whole runes", head)
	}
}

// TestAgentWriterPreservesRunesAcrossChunks is the same hazard on the streaming
// path: a long non-ASCII line must survive chunking intact.
func TestAgentWriterPreservesRunesAcrossChunks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	bus, err := Open(path, "run")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer bus.Close()

	// A single line well over maxChunk, entirely multi-byte runes.
	line := strings.Repeat("世", maxChunk) // 3x maxChunk bytes
	w := bus.AgentWriter("mod", RoleDev)
	if _, err := w.Write([]byte(line + "\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	all, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	var rebuilt strings.Builder
	for _, ev := range all {
		var out AgentOutput
		if err := ev.Decode(&out); err != nil {
			t.Fatalf("Decode: %v", err)
		}
		if !utf8.ValidString(out.Text) {
			t.Fatalf("chunk %d is not valid UTF-8", ev.Seq)
		}
		rebuilt.WriteString(out.Text)
	}
	if rebuilt.String() != line {
		t.Errorf("reassembled text differs from what the agent wrote (%d vs %d bytes)",
			rebuilt.Len(), len(line))
	}
}

func TestReadAllMissingFile(t *testing.T) {
	got, err := ReadAll(filepath.Join(t.TempDir(), "nope.jsonl"))
	if err != nil {
		t.Errorf("err = %v, want nil for a missing file", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestSlug(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Add login endpoint", "add_login_endpoint"},
		{"  ...  ", ""},
		{"UPPER-case_Mix 123", "upper_case_mix_123"},
		{strings.Repeat("a", 50), strings.Repeat("a", 32)},
		{"", ""},
	}
	for _, c := range cases {
		if got := Slug(c.in); got != c.want {
			t.Errorf("Slug(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestNewRunID pins the identity that replaced the old hardcoded TaskID, which
// labelled every run "cti_modular_self_healing".
func TestNewRunID(t *testing.T) {
	ts := time.Date(2026, 7, 17, 10, 15, 33, 0, time.UTC)

	if got, want := NewRunID("Add login", ts), "add_login-20260717T101533Z"; got != want {
		t.Errorf("NewRunID = %q, want %q", got, want)
	}
	// An unlabelled run still gets a usable, unique id.
	if got, want := NewRunID("", ts), "run-20260717T101533Z"; got != want {
		t.Errorf("NewRunID(empty) = %q, want %q", got, want)
	}
}
