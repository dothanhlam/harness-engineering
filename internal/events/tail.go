package events

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"time"
)

// DefaultPollInterval is how often a Tailer re-stats the log. Polling keeps the
// package stdlib-only (no fsnotify) and behaves the same on every platform.
const DefaultPollInterval = 150 * time.Millisecond

// maxLineBytes is the largest single event record a reader will accept. It sits
// far above any capped payload so it is only ever reached by a corrupt file.
const maxLineBytes = 32 << 20

// Tailer streams events from a JSONL log, replaying what is already there and
// then following the file as it grows. The same code path serves "watch a live
// run" and "review a finished one", since a completed log simply stops growing.
type Tailer struct {
	Path string
	Poll time.Duration // zero means DefaultPollInterval

	offset int64
}

// NewTailer returns a Tailer reading path from the beginning.
func NewTailer(path string) *Tailer {
	return &Tailer{Path: path, Poll: DefaultPollInterval}
}

// Follow calls fn for each event, in order, until ctx is cancelled. It returns
// nil on cancellation and never returns on its own: a log that has stopped
// growing is indistinguishable from an idle one (a slow agent can be quiet for
// minutes), so ending the stream is the caller's decision — typically on seeing
// a KindRunFinished event.
//
// A missing file is not an error; Follow waits for it to appear, so a monitor
// can be started before the run.
func (t *Tailer) Follow(ctx context.Context, fn func(Event)) error {
	poll := t.Poll
	if poll <= 0 {
		poll = DefaultPollInterval
	}
	ticker := time.NewTicker(poll)
	defer ticker.Stop()

	for {
		if err := t.drain(fn); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// Replay dispatches every event already in the log and advances the read
// position, so a subsequent Follow on the same Tailer reports only events
// appended afterwards. Use it to render a run's existing history before
// following live, without the two passes overlapping.
func (t *Tailer) Replay(fn func(Event)) error {
	return t.drain(fn)
}

// drain reads and dispatches every complete line written since the last call.
func (t *Tailer) drain(fn func(Event)) error {
	f, err := os.Open(t.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // not created yet
		}
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	// A shrunken file means a new run truncated the log; start over.
	if info.Size() < t.offset {
		t.offset = 0
	}
	if info.Size() == t.offset {
		return nil
	}

	if _, err := f.Seek(t.offset, io.SeekStart); err != nil {
		return err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	// Consume up to the last newline only. Anything after it is a line still
	// being written, or one torn by a crashed run; either way it is not ours to
	// parse yet.
	end := bytes.LastIndexByte(data, '\n')
	if end < 0 {
		return nil
	}
	t.offset += int64(end + 1)

	for _, line := range bytes.Split(data[:end], []byte{'\n'}) {
		if ev, ok := parseLine(line); ok {
			fn(ev)
		}
	}
	return nil
}

// ReadAll loads every event currently in path. Used for one-shot snapshots.
func ReadAll(path string) ([]Event, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []Event
	scanner := bufio.NewScanner(f)
	// Payloads are capped at emit time (see MaxErrorText), but this is the last
	// line of defence: a single over-long record must not make bufio.Scanner
	// fail the whole read and report an empty run.
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	for scanner.Scan() {
		if ev, ok := parseLine(scanner.Bytes()); ok {
			out = append(out, ev)
		}
	}
	if err := scanner.Err(); err != nil {
		return out, err
	}
	return out, nil
}

// parseLine decodes one line, reporting false for blank or malformed input.
// A malformed line is skipped rather than fatal: a monitor must survive a run
// that died mid-write.
func parseLine(line []byte) (Event, bool) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return Event{}, false
	}
	var ev Event
	if err := json.Unmarshal(line, &ev); err != nil {
		return Event{}, false
	}
	return ev, true
}
