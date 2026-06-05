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

// AgentSpec defines a pluggable CLI agent with dynamic command templates and environment injection.
type AgentSpec struct {
	Agent       string            `json:"agent"`
	ModelName   string            `json:"model_name,omitempty"`
	CmdTemplate []string          `json:"cmd_template"`
	Env         map[string]string `json:"env,omitempty"`
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
	var usage TokenUsage
	if len(a.CmdTemplate) == 0 {
		return "", usage, fmt.Errorf("agent %s has no cmd_template defined", a.Agent)
	}

	var finalArgs []string
	for _, arg := range a.CmdTemplate {
		arg = strings.ReplaceAll(arg, "{prompt}", prompt)
		arg = strings.ReplaceAll(arg, "{model}", a.ModelName)
		finalArgs = append(finalArgs, arg)
	}

	cmd := exec.CommandContext(ctx, a.Agent, finalArgs...)

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

	// Attempt to parse token usage from stderr (primarily for Ollama --verbose)
	usage.PromptTokens = extractOllamaTokenMetric(stderr.String(), "prompt eval count:")
	usage.EvalTokens = extractOllamaTokenMetric(stderr.String(), "eval count:")

	return strings.TrimSpace(out.String()), usage, nil
}

func extractOllamaTokenMetric(stderr string, prefix string) int {
	// Look for lines like: "prompt eval count:    42 token(s)"
	lines := strings.Split(stderr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			re := regexp.MustCompile(`[0-9]+`)
			match := re.FindString(line)
			if match != "" {
				val, err := strconv.Atoi(match)
				if err == nil {
					return val
				}
			}
		}
	}
	return 0
}

// Clone creates a deep copy of an AgentSpec, safe for concurrent modification.
func (a *AgentSpec) Clone() AgentSpec {
	clone := *a
	clone.CmdTemplate = make([]string, len(a.CmdTemplate))
	copy(clone.CmdTemplate, a.CmdTemplate)
	if a.Env != nil {
		clone.Env = make(map[string]string)
		for k, v := range a.Env {
			clone.Env[k] = v
		}
	}
	return clone
}
