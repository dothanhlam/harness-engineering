package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// llama.cpp prints performance timings to stderr on exit, e.g.:
//
//	llama_perf_context_print: prompt eval time =  123.45 ms /   42 tokens (...)
//	llama_perf_context_print:        eval time = 1234.56 ms /  100 runs   (...)
//
// promptEvalRe captures the prompt token count; evalRunsRe captures the
// generated (eval) token count.
var (
	promptEvalRe = regexp.MustCompile(`prompt eval time =.*?/\s*(\d+)\s+tokens`)
	evalRunsRe   = regexp.MustCompile(`\beval time =.*?/\s*(\d+)\s+runs`)
)

// parseLlamaTokenUsage extracts prompt/eval token counts from llama.cpp's
// stderr timing output. Returns a zero-valued TokenUsage if timings are absent.
func parseLlamaTokenUsage(stderr string) TokenUsage {
	var usage TokenUsage
	if m := promptEvalRe.FindStringSubmatch(stderr); m != nil {
		usage.PromptTokens, _ = strconv.Atoi(m[1])
	}
	if m := evalRunsRe.FindStringSubmatch(stderr); m != nil {
		usage.EvalTokens, _ = strconv.Atoi(m[1])
	}
	return usage
}

// AgentSpec defines a pluggable CLI agent with dynamic command templates and environment injection.
type AgentSpec struct {
	Agent          string            `json:"agent"`
	ModelName      string            `json:"model_name,omitempty"`
	CmdTemplate    []string          `json:"cmd_template,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	ContextWindow  int               `json:"-"`
	FlashAttention bool              `json:"-"`
}

// TokenUsage holds extracted token metrics (if emitted by the agent).
type TokenUsage struct {
	PromptTokens int
	EvalTokens   int
}

// Execute runs the agent CLI with the given prompt, replacing template tokens dynamically.
// Supported tokens: {prompt}, {model}
func (a *AgentSpec) Execute(prompt string) (string, TokenUsage, error) {
	return a.ExecuteWithContext(context.Background(), prompt)
}

// ExecuteWithContext runs the agent CLI with a given context, for timeouts and cancellation.
func (a *AgentSpec) ExecuteWithContext(ctx context.Context, prompt string) (string, TokenUsage, error) {
	return a.executeViaCLI(ctx, prompt)
}


// ─── CLI Mode (local development) ───────────────────────────────────────────

func (a *AgentSpec) executeViaCLI(ctx context.Context, prompt string) (string, TokenUsage, error) {
	var usage TokenUsage

	var finalArgs []string
	agentCmd := a.Agent

	if a.Agent == "llama_cpp" {
		agentCmd = "llama-completion"

		modelPath := a.ModelName
		if strings.HasPrefix(modelPath, "hf://") {
			finalArgs = append(finalArgs, "-hf", strings.TrimPrefix(modelPath, "hf://"))
		} else {
			if strings.HasPrefix(modelPath, "~/") {
				homeDir, err := os.UserHomeDir()
				if err == nil {
					modelPath = homeDir + modelPath[1:]
				}
			}
			finalArgs = append(finalArgs, "-m", modelPath)
		}
		if a.ContextWindow > 0 {
			finalArgs = append(finalArgs, "-c", strconv.Itoa(a.ContextWindow))
		}
		if a.FlashAttention {
			finalArgs = append(finalArgs, "--flash-attn", "on")
		}
		finalArgs = append(finalArgs, "--no-display-prompt", "-p", prompt)
	} else {
		if len(a.CmdTemplate) == 0 {
			return "", usage, fmt.Errorf("agent %s has no cmd_template defined", a.Agent)
		}
		for _, arg := range a.CmdTemplate {
			arg = strings.ReplaceAll(arg, "{prompt}", prompt)
			arg = strings.ReplaceAll(arg, "{model}", a.ModelName)
			finalArgs = append(finalArgs, arg)
		}
	}

	cmd := exec.CommandContext(ctx, agentCmd, finalArgs...)

	if len(a.Env) > 0 {
		envVars := os.Environ()
		for k, v := range a.Env {
			v = strings.ReplaceAll(v, "{prompt}", prompt)
			v = strings.ReplaceAll(v, "{model}", a.ModelName)
			envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = envVars
	}

	var out bytes.Buffer
	var stderr bytes.Buffer

	// Bind directly to os terminal while still capturing output for returns
	cmd.Stdout = io.MultiWriter(os.Stdout, &out)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

	if err := cmd.Run(); err != nil {
		return "", usage, fmt.Errorf("%s CLI error: %v, stderr: %s", a.Agent, err, stderr.String())
	}

	// llama.cpp emits token timings to stderr; parse them into telemetry.
	if a.Agent == "llama_cpp" {
		usage = parseLlamaTokenUsage(stderr.String())
	}

	return strings.TrimSpace(out.String()), usage, nil
}


// Clone creates a deep copy of an AgentSpec, safe for concurrent modification.
func (a *AgentSpec) Clone() AgentSpec {
	clone := *a
	if len(a.CmdTemplate) > 0 {
		clone.CmdTemplate = make([]string, len(a.CmdTemplate))
		copy(clone.CmdTemplate, a.CmdTemplate)
	}
	if a.Env != nil {
		clone.Env = make(map[string]string)
		for k, v := range a.Env {
			clone.Env[k] = v
		}
	}
	return clone
}
