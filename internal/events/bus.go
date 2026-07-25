package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unicode/utf8"
)

// maxChunk bounds the Text of a single AgentOutput event. A local llama.cpp
// model can emit a very long line (or none at all), and an unbounded line would
// mean an unbounded buffer here and an unbounded record for every reader.
const maxChunk = 3500

// MaxErrorText bounds error text carried in a payload, such as a failing
// `go test -v` dump. Left uncapped, one QA failure can produce a single JSONL
// line of many megabytes, which readers must then buffer whole; the head of a
// compiler or test error is what a monitor actually displays.
const MaxErrorText = 16 << 10

// runeSafeCut returns the largest n <= max such that s[:n] ends on a complete
// UTF-8 rune, so splitting never cuts a multi-byte character in half (which
// json.Marshal would replace with U+FFFD, silently corrupting the text).
func runeSafeCut[T ~string | ~[]byte](s T, max int) int {
	if len(s) <= max {
		return len(s)
	}
	n := max
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	if n == 0 { // pathological: no boundary found within max
		return max
	}
	return n
}

// TruncateText caps s at max bytes on a rune boundary, noting what was dropped.
func TruncateText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	n := runeSafeCut(s, max)
	return s[:n] + fmt.Sprintf("\n… [truncated: %d of %d bytes shown]", n, len(s))
}

// Bus appends events to a JSONL file. It is safe for concurrent use: the
// parallel epic path emits from many goroutines at once.
//
// A nil *Bus is a valid no-op receiver. Callers that could not open a log pass
// nil and every emit becomes a cheap nothing, so monitoring can never take the
// pipeline down with it.
type Bus struct {
	mu    sync.Mutex
	f     *os.File
	runID string
	seq   int
}

// Open truncates path and returns a Bus writing to it. The file holds exactly
// one run; a monitor tailing across runs detects the truncation and re-reads.
func Open(path, runID string) (*Bus, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	return &Bus{f: f, runID: runID}, nil
}

// RunID returns the identifier stamped on every event ("" for a nil Bus).
func (b *Bus) RunID() string {
	if b == nil {
		return ""
	}
	return b.runID
}

// Emit appends one event. task scopes the event to a sub-task (see Event.Task)
// and may be empty. A payload that fails to marshal is dropped rather than
// failing the run.
//
// Writes are unbuffered, so events survive the log.Fatalf and os.Exit paths in
// the pipeline without an explicit flush.
func (b *Bus) Emit(kind Kind, task string, payload any) {
	if b == nil {
		return
	}

	var raw json.RawMessage
	if payload != nil {
		if data, err := json.Marshal(payload); err == nil {
			raw = data
		}
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.seq++
	line, err := json.Marshal(Event{
		Seq:     b.seq,
		RunID:   b.runID,
		TS:      time.Now().UTC(),
		Kind:    kind,
		Task:    task,
		Payload: raw,
	})
	if err != nil {
		return
	}
	_, _ = b.f.Write(append(line, '\n'))
}

// Close closes the underlying file.
func (b *Bus) Close() error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.f.Close()
}

// AgentWriter returns an io.Writer that mirrors an agent's stdout into the
// stream as line-oriented AgentOutput events, tagged with task and role.
//
// Returns nil for a nil Bus, which callers must treat as "no sink".
func (b *Bus) AgentWriter(task, role string) io.Writer {
	if b == nil {
		return nil
	}
	return &agentWriter{bus: b, task: task, role: role}
}

// agentWriter splits a byte stream into lines and emits one event per line.
//
// Write never reports an error. It is attached to an io.MultiWriter alongside
// os.Stdout, and io.MultiWriter aborts the whole write on the first error — so
// failing here would silently cut off the agent's terminal output.
type agentWriter struct {
	bus  *Bus
	task string
	role string

	mu  sync.Mutex
	buf []byte
}

func (w *agentWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		w.emit(bytes.TrimRight(w.buf[:i], "\r"))
		w.buf = w.buf[i+1:]
	}

	// Output with no newline in sight would otherwise buffer without bound.
	for len(w.buf) >= maxChunk {
		n := runeSafeCut(w.buf, maxChunk)
		w.emit(w.buf[:n])
		w.buf = w.buf[n:]
	}
	return len(p), nil
}

// Flush emits any trailing bytes not terminated by a newline.
func (w *agentWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buf) > 0 {
		w.emit(w.buf)
		w.buf = w.buf[:0]
	}
}

// emit splits over-long lines into several events. Callers hold w.mu.
func (w *agentWriter) emit(b []byte) {
	for len(b) > maxChunk {
		n := runeSafeCut(b, maxChunk)
		w.bus.Emit(KindAgentOutput, w.task, AgentOutput{Role: w.role, Text: string(b[:n])})
		b = b[n:]
	}
	w.bus.Emit(KindAgentOutput, w.task, AgentOutput{Role: w.role, Text: string(b)})
}

// Flush drains any buffered partial line from a writer returned by AgentWriter.
// It is a no-op for any other writer, including nil.
func Flush(w io.Writer) {
	if f, ok := w.(interface{ Flush() }); ok {
		f.Flush()
	}
}
