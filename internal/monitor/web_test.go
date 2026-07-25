package monitor

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dothanhlam/harness-engineering/internal/events"
)

func TestWeb_IndexServed(t *testing.T) {
	srv := httptest.NewServer(newMux(filepath.Join(t.TempDir(), "e.jsonl")))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	body := readBody(t, resp)
	if !strings.Contains(body, "harness monitor") || !strings.Contains(body, "EventSource") {
		t.Errorf("index page missing expected markers")
	}
}

func TestWeb_StateSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "e.jsonl")
	bus, _ := events.Open(path, "run-x")
	bus.Emit(events.KindRunStarted, "", events.RunStarted{Mode: "task", Task: "demo"})
	bus.Emit(events.KindStageChanged, "demo", events.StageChanged{Stage: "DEV_CODING", MaxRetries: 3})
	bus.Close()

	srv := httptest.NewServer(newMux(path))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/state")
	if err != nil {
		t.Fatalf("GET /api/state: %v", err)
	}
	defer resp.Body.Close()

	var got []events.Event
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 || got[0].Kind != events.KindRunStarted || got[0].RunID != "run-x" {
		t.Fatalf("snapshot = %+v", got)
	}
}

// TestWeb_StateEmpty: a missing log yields an empty JSON array, not null or 500.
func TestWeb_StateEmpty(t *testing.T) {
	srv := httptest.NewServer(newMux(filepath.Join(t.TempDir(), "missing.jsonl")))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/state")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if got := strings.TrimSpace(readBody(t, resp)); got != "[]" {
		t.Errorf("empty state = %q, want []", got)
	}
}

// TestWeb_SSEStream replays existing events then delivers a live one, proving the
// stream both catches up and follows.
func TestWeb_SSEStream(t *testing.T) {
	path := filepath.Join(t.TempDir(), "e.jsonl")
	bus, _ := events.Open(path, "run-sse")
	defer bus.Close()
	bus.Emit(events.KindRunStarted, "", events.RunStarted{Mode: "task", Task: "demo"})

	srv := httptest.NewServer(newMux(path))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	frames := make(chan sseFrame, 8)
	go readFrames(resp.Body, frames)

	// First frame: the replayed run_started.
	got := waitFrame(t, frames)
	if got.id != "1" {
		t.Errorf("first frame id = %q, want 1", got.id)
	}
	var ev events.Event
	if err := json.Unmarshal([]byte(got.data), &ev); err != nil {
		t.Fatalf("frame data not an Event: %v (%q)", err, got.data)
	}
	if ev.Kind != events.KindRunStarted {
		t.Errorf("first event kind = %q", ev.Kind)
	}

	// A live event appended after connect must arrive on the same stream.
	bus.Emit(events.KindStageChanged, "demo", events.StageChanged{Stage: "DEV_CODING", MaxRetries: 3})
	got = waitFrame(t, frames)
	if got.id != "2" {
		t.Errorf("live frame id = %q, want 2", got.id)
	}
	if err := json.Unmarshal([]byte(got.data), &ev); err != nil || ev.Kind != events.KindStageChanged {
		t.Errorf("live frame = %q (%v)", got.data, err)
	}
}

// ── SSE parsing helpers ───────────────────────────────────────────

type sseFrame struct{ id, data string }

func readFrames(r io.Reader, out chan<- sseFrame) {
	sc := bufio.NewScanner(r)
	var cur sseFrame
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			if cur.data != "" {
				out <- cur
			}
			cur = sseFrame{}
		case strings.HasPrefix(line, "id: "):
			cur.id = strings.TrimPrefix(line, "id: ")
		case strings.HasPrefix(line, "data: "):
			cur.data = strings.TrimPrefix(line, "data: ")
		}
	}
	close(out)
}

func waitFrame(t *testing.T, ch chan sseFrame) sseFrame {
	t.Helper()
	select {
	case f, ok := <-ch:
		if !ok {
			t.Fatal("SSE stream closed before a frame arrived")
		}
		return f
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for an SSE frame")
		return sseFrame{}
	}
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	var b strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		b.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return b.String()
}
