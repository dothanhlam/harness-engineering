package events

import (
	"context"
	"time"

	"github.com/dothanhlam/harness-engineering/internal/agent"
)

// Run invokes an agent, bracketing the call with AgentStarted/AgentDone events
// and mirroring its stdout as AgentOutput. It is the single wrapper every agent
// call site in the pipeline goes through, so observability stays uniform and
// each site stays a one-liner.
//
// A nil Bus degrades to a plain spec.Execute.
func Run(b *Bus, task, role string, spec *agent.AgentSpec, prompt string) (string, agent.TokenUsage, error) {
	return RunWithContext(context.Background(), b, task, role, spec, prompt)
}

// RunWithContext is Run with a caller-supplied context, for the timeout-bounded
// llama.cpp paths.
func RunWithContext(ctx context.Context, b *Bus, task, role string, spec *agent.AgentSpec, prompt string) (string, agent.TokenUsage, error) {
	if b == nil {
		return spec.ExecuteWithContext(ctx, prompt)
	}

	b.Emit(KindAgentStarted, task, AgentStarted{
		Role:  role,
		Agent: spec.Agent,
		Model: spec.ModelName,
	})

	// Clone rather than set Sink on the caller's spec: the epic path shares one
	// *AgentSpec across concurrent goroutines, so mutating it would be a race.
	local := spec.Clone()
	local.Sink = b.AgentWriter(task, role)

	start := time.Now()
	out, usage, err := local.ExecuteWithContext(ctx, prompt)
	Flush(local.Sink)

	done := AgentDone{
		Role:         role,
		PromptTokens: usage.PromptTokens,
		EvalTokens:   usage.EvalTokens,
		DurationSecs: time.Since(start).Seconds(),
	}
	if err != nil {
		done.Error = err.Error()
	}
	b.Emit(KindAgentDone, task, done)

	return out, usage, err
}
