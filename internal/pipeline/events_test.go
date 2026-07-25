package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dothanhlam/harness-engineering/internal/events"
)

// busInDir opens an event log inside a temp dir and chdirs there, so the
// pipeline's relative paths resolve under the test's own sandbox.
func busInDir(t *testing.T) (*events.Bus, string) {
	t.Helper()

	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	logPath := filepath.Join(dir, "events.jsonl")
	bus, err := events.Open(logPath, "test-run")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = bus.Close() })
	return bus, logPath
}

// kinds returns the events in the log matching kind.
func kinds(t *testing.T, path string, kind events.Kind) []events.Event {
	t.Helper()
	all, err := events.ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	var out []events.Event
	for _, ev := range all {
		if ev.Kind == kind {
			out = append(out, ev)
		}
	}
	return out
}

// TestParseAndWriteGeneratedFiles_EmitsWrites covers full-file and patch output.
func TestParseAndWriteGeneratedFiles_EmitsWrites(t *testing.T) {
	bus, logPath := busInDir(t)

	output := "### FILE: workspace/demo/demo.go\n```\npackage demo\n\nfunc Add(a, b int) int { return a + b }\n```\n"
	parseAndWriteGeneratedFiles(output, "workspace/demo", "demo", bus)

	if _, err := os.Stat("workspace/demo/demo.go"); err != nil {
		t.Fatalf("expected file on disk: %v", err)
	}

	written := kinds(t, logPath, events.KindFileWritten)
	if len(written) != 1 {
		t.Fatalf("got %d file_written events, want 1", len(written))
	}
	var fw events.FileWritten
	if err := written[0].Decode(&fw); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if fw.Path != "workspace/demo/demo.go" || fw.Action != "write" {
		t.Errorf("FileWritten = %+v", fw)
	}
	if written[0].Task != "demo" {
		t.Errorf("Task = %q, want demo", written[0].Task)
	}

	// A SEARCH/REPLACE block against the file just written reports action=patch.
	patch := "### FILE: workspace/demo/demo.go\n<<<<\nreturn a + b\n====\nreturn b + a\n>>>>\n"
	parseAndWriteGeneratedFiles(patch, "workspace/demo", "demo", bus)

	written = kinds(t, logPath, events.KindFileWritten)
	if len(written) != 2 {
		t.Fatalf("got %d file_written events, want 2", len(written))
	}
	if err := written[1].Decode(&fw); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if fw.Action != "patch" {
		t.Errorf("Action = %q, want patch", fw.Action)
	}
}

// TestParseAndWriteGeneratedFiles_EmitsRejection covers the security boundary:
// a generated path escaping the target folder must be refused *and* reported,
// since the QA audit only scans the target folder and would never see a file
// written outside it.
func TestParseAndWriteGeneratedFiles_EmitsRejection(t *testing.T) {
	bus, logPath := busInDir(t)

	output := "### FILE: workspace/other/evil.go\n```\npackage evil\n```\n"
	parseAndWriteGeneratedFiles(output, "workspace/demo", "demo", bus)

	if _, err := os.Stat("workspace/other/evil.go"); !os.IsNotExist(err) {
		t.Fatal("escaping file was written to disk")
	}
	if got := kinds(t, logPath, events.KindFileWritten); len(got) != 0 {
		t.Errorf("got %d file_written events, want 0", len(got))
	}

	rejected := kinds(t, logPath, events.KindFileRejected)
	if len(rejected) != 1 {
		t.Fatalf("got %d file_rejected events, want 1", len(rejected))
	}
	var fr events.FileRejected
	if err := rejected[0].Decode(&fr); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if fr.Path != "workspace/other/evil.go" || fr.Target != "workspace/demo" {
		t.Errorf("FileRejected = %+v", fr)
	}
}

// TestParseAndWriteGeneratedFiles_NilBus confirms the parser still writes code
// when no event log is open.
func TestParseAndWriteGeneratedFiles_NilBus(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()

	output := "### FILE: workspace/demo/demo.go\n```\npackage demo\n```\n"
	parseAndWriteGeneratedFiles(output, "workspace/demo", "demo", nil)

	if _, err := os.Stat("workspace/demo/demo.go"); err != nil {
		t.Fatalf("expected file on disk with a nil bus: %v", err)
	}
}
