package events

import (
	"github.com/dothanhlam/harness-engineering/internal/telemetry"
)

// EmitRunFinished reports the terminal outcome of a run, carrying the
// authoritative totals from the tracker. Every exit path goes through here so
// the stream's last event always has the same shape.
//
// Call it *before* log.Fatalf or os.Exit on the abort paths: those skip
// deferred cleanup. Bus writes are unbuffered, so the event still lands.
//
// Totals come from the tracker snapshot, so callers that want an accurate
// LinesOfCode should call tracker.Finalize first; the abort paths legitimately
// report zero.
func EmitRunFinished(b *Bus, task string, tracker *telemetry.Tracker, success bool, reason string, failedTasks ...string) {
	if b == nil {
		return
	}
	snap := tracker.Snapshot()
	b.Emit(KindRunFinished, task, RunFinished{
		Success:            success,
		Reason:             reason,
		FailedTasks:        failedTasks,
		DurationSeconds:    snap.TotalDurationSeconds,
		TotalPromptTokens:  snap.TotalPromptTokens,
		TotalEvalTokens:    snap.TotalEvalTokens,
		TotalRetriesUsed:   snap.TotalRetriesUsed,
		CodeHealingSuccess: snap.CodeHealingSuccess,
		LinesOfCode:        snap.LinesOfCodeGenerated,
	})
}
