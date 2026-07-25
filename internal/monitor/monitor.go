// Package monitor renders a harness event stream to a terminal.
//
// It is the first consumer of internal/events and is deliberately thin: a
// stateless per-event formatter plus a follow loop over events.Tailer. The web
// UI (SSE) and TUI planned for later steps reuse the same tailer and can reuse
// FormatEvent for their text panes.
package monitor

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dothanhlam/harness-engineering/internal/events"
)

// Options configures a follow session.
type Options struct {
	// Once replays the events already in the log and returns, instead of
	// following the file live. Use it to review a finished run.
	Once bool
	// ShowOutput includes mirrored agent stdout (KindAgentOutput). It is the
	// most useful live signal for a slow local model but also the noisiest, so
	// callers can turn it off for a high-level view.
	ShowOutput bool
	// Color forces ANSI styling on or off. When nil, styling follows whether
	// the writer is a terminal.
	Color *bool
}

// Follow renders events from path to out until the run finishes or ctx is
// cancelled. It replays what is already in the log, then (unless Once) tails it
// live, stopping at the single KindRunFinished that terminates every run.
//
// A log that has not appeared yet is not an error: Follow waits for it, so a
// monitor may be started before the run.
func Follow(ctx context.Context, path string, out io.Writer, opts Options) error {
	r := newRenderer(out, resolveColor(opts.Color, out))

	if opts.Once {
		all, err := events.ReadAll(path)
		if err != nil {
			return err
		}
		for _, ev := range all {
			r.write(ev, opts.ShowOutput)
		}
		return nil
	}

	tl := events.NewTailer(path)

	// Render whatever history is already on disk, but do NOT treat a run_finished
	// found here as a reason to stop: the log is only truncated when the next run
	// begins, so a completed run persists between runs. Stopping on it would make
	// a monitor started before the next run replay the stale run and exit before
	// the new one appears. Use --once to replay-and-exit a finished run instead.
	if err := tl.Replay(func(ev events.Event) { r.write(ev, opts.ShowOutput) }); err != nil {
		return err
	}

	// Follow never returns on its own — a quiet log is indistinguishable from a
	// slow agent — so a run_finished observed live cancels this derived context.
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	// Cancelling the context only ends the next poll. A single drain delivers a
	// whole batch of buffered lines before rechecking ctx, so once run_finished
	// is seen this flag also suppresses anything queued behind it. The tailer
	// invokes the callback from one goroutine, so a plain bool is safe.
	done := false
	return tl.Follow(ctx, func(ev events.Event) {
		if done {
			return
		}
		r.write(ev, opts.ShowOutput)
		if ev.Kind == events.KindRunFinished {
			done = true
			stop()
		}
	})
}

// resolveColor honours an explicit override, else enables ANSI only when out is
// a character device (a terminal) — never when redirected to a file or pipe.
func resolveColor(override *bool, out io.Writer) bool {
	if override != nil {
		return *override
	}
	f, ok := out.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// ── rendering ────────────────────────────────────────────────────────────────

const (
	ansiReset  = "\033[0m"
	ansiDim    = "\033[2m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiBlue   = "\033[34m"
	ansiCyan   = "\033[36m"
)

type renderer struct {
	out   io.Writer
	color bool
}

func newRenderer(out io.Writer, color bool) *renderer {
	return &renderer{out: out, color: color}
}

func (r *renderer) write(ev events.Event, showOutput bool) {
	line, ok := FormatEvent(ev, r.color, showOutput)
	if !ok {
		return
	}
	fmt.Fprintln(r.out, line)
}

// FormatEvent renders one event as a single terminal line. It returns ok=false
// for events that should not print (agent output when showOutput is false, or an
// unknown kind). color toggles ANSI styling; showOutput includes mirrored agent
// stdout.
//
// It is a pure function of the event so later UIs can reuse it for text panes.
func FormatEvent(ev events.Event, color, showOutput bool) (string, bool) {
	ts := ev.TS.Local().Format("15:04:05")
	c := colorizer{color}

	glyph, body, ok := describe(ev, c, showOutput)
	if !ok {
		return "", false
	}

	line := fmt.Sprintf("%s %s %s", c.paint(ansiDim, ts), glyph, body)
	if ev.Task != "" {
		line += "  " + c.paint(ansiDim, "["+ev.Task+"]")
	}
	return line, true
}

// describe returns the glyph and message body for an event, or ok=false to skip.
func describe(ev events.Event, c colorizer, showOutput bool) (glyph, body string, ok bool) {
	switch ev.Kind {
	case events.KindRunStarted:
		var p events.RunStarted
		_ = ev.Decode(&p)
		roles := make([]string, 0, len(p.Agents))
		for _, role := range []string{events.RoleBA, events.RoleDev, events.RoleDevOps} {
			if a, has := p.Agents[role]; has {
				roles = append(roles, fmt.Sprintf("%s=%s", role, a))
			}
		}
		target := p.Task
		if p.Epic != "" {
			target = p.Epic
		}
		head := strings.ToUpper(p.Mode) + " run"
		if target != "" {
			head += " " + c.paint(ansiCyan, target)
		}
		return c.paint(ansiCyan, "▶"),
			fmt.Sprintf("%s  %s", head, c.paint(ansiDim, strings.Join(roles, "  "))),
			true

	case events.KindStageChanged:
		var p events.StageChanged
		_ = ev.Decode(&p)
		msg := c.paint(ansiBlue, p.Stage)
		if p.MaxRetries > 1 {
			msg += c.paint(ansiDim, fmt.Sprintf("  attempt %d/%d", p.Retry+1, p.MaxRetries))
		}
		return c.paint(ansiBlue, "▸"), msg, true

	case events.KindAgentStarted:
		var p events.AgentStarted
		_ = ev.Decode(&p)
		who := p.Agent
		if p.Model != "" {
			who += " (" + p.Model + ")"
		}
		return "⟳", fmt.Sprintf("%s  %s", p.Role, c.paint(ansiDim, who)), true

	case events.KindAgentOutput:
		if !showOutput {
			return "", "", false
		}
		var p events.AgentOutput
		_ = ev.Decode(&p)
		return c.paint(ansiDim, "·"), c.paint(ansiDim, p.Text), true

	case events.KindAgentDone:
		var p events.AgentDone
		_ = ev.Decode(&p)
		if p.Error != "" {
			return c.paint(ansiRed, "✗"),
				fmt.Sprintf("%s failed  %s", p.Role, c.paint(ansiRed, oneLine(p.Error))), true
		}
		return c.paint(ansiGreen, "✓"),
			fmt.Sprintf("%s done  %s", p.Role,
				c.paint(ansiDim, fmt.Sprintf("%d→%d tok  %.1fs", p.PromptTokens, p.EvalTokens, p.DurationSecs))),
			true

	case events.KindQAResult:
		var p events.QAResult
		_ = ev.Decode(&p)
		if p.Passed {
			return c.paint(ansiGreen, "✓"), c.paint(ansiGreen, "QA passed")+"  "+c.paint(ansiDim, p.Target), true
		}
		reason := firstNonEmpty(p.AuditError, p.TestError)
		return c.paint(ansiRed, "✗"),
			fmt.Sprintf("%s  attempt %d/%d  %s", c.paint(ansiRed, "QA failed"), p.Attempt, p.MaxRetries, c.paint(ansiDim, oneLine(reason))),
			true

	case events.KindRetry:
		var p events.Retry
		_ = ev.Decode(&p)
		return c.paint(ansiYellow, "↻"),
			c.paint(ansiYellow, fmt.Sprintf("self-heal retry (attempt %d/%d failed)", p.Attempt, p.MaxRetries)), true

	case events.KindDelegation:
		var p events.Delegation
		_ = ev.Decode(&p)
		return c.paint(ansiYellow, "⇄"),
			c.paint(ansiYellow, fmt.Sprintf("delegating to BA (cycle %d/%d)", p.Cycle+1, p.MaxCycle)), true

	case events.KindFileWritten:
		var p events.FileWritten
		_ = ev.Decode(&p)
		return c.paint(ansiGreen, "＋"), fmt.Sprintf("%s %s", p.Action, p.Path), true

	case events.KindFileRejected:
		var p events.FileRejected
		_ = ev.Decode(&p)
		return c.paint(ansiRed, "⛔"),
			fmt.Sprintf("%s  %s", c.paint(ansiRed, "rejected "+p.Path), c.paint(ansiDim, p.Reason)), true

	case events.KindHITLPrompt:
		var p events.HITLPrompt
		_ = ev.Decode(&p)
		return c.paint(ansiYellow, "⏸"),
			c.paint(ansiYellow, fmt.Sprintf("awaiting approval (auto-yes in %ds) — answer in the run's terminal", p.TimeoutSeconds)), true

	case events.KindHITLResolved:
		var p events.HITLResolved
		_ = ev.Decode(&p)
		verb := "approved"
		if !p.Approved {
			verb = "rejected"
		}
		if p.Auto {
			verb += " (auto)"
		}
		return c.paint(ansiCyan, "▸"), "HITL "+verb, true

	case events.KindRunFinished:
		var p events.RunFinished
		_ = ev.Decode(&p)
		glyph = c.paint(ansiGreen, "■")
		head := c.paint(ansiGreen, "run finished · success")
		if !p.Success {
			glyph = c.paint(ansiRed, "■")
			head = c.paint(ansiRed, "run finished · FAILED")
		}
		stats := fmt.Sprintf("%.1fs  %d→%d tok  %d retries  %d LOC",
			p.DurationSeconds, p.TotalPromptTokens, p.TotalEvalTokens, p.TotalRetriesUsed, p.LinesOfCode)
		body = head + "  " + c.paint(ansiDim, stats)
		if p.Reason != "" {
			body += "  " + c.paint(ansiDim, "— "+oneLine(p.Reason))
		}
		return glyph, body, true

	default:
		return "", "", false
	}
}

// colorizer applies ANSI codes only when enabled.
type colorizer struct{ on bool }

func (c colorizer) paint(code, s string) string {
	if !c.on {
		return s
	}
	return code + s + ansiReset
}

// oneLine collapses a multi-line error to the most useful single line so it fits
// on one terminal row; the full text lives in the log and workspace/qa_error.log.
//
// It prefers the first real content line, skipping go's "# package" markers that
// head `go build`/`go test` output — the actual compiler error is the line after
// the marker. If a marker is all there is, it is still returned.
func oneLine(s string) string {
	var first string
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if first == "" {
			first = t
		}
		if !strings.HasPrefix(t, "# ") {
			return t
		}
	}
	return first
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
