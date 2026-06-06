package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
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
// If OLLAMA_HOST is set and the agent is "ollama", uses the HTTP API instead of CLI subprocess.
func (a *AgentSpec) Execute(prompt string) (string, TokenUsage, error) {
	return a.ExecuteWithContext(context.Background(), prompt)
}

// ExecuteWithContext runs the agent CLI with a given context, for timeouts and cancellation.
// Automatically routes to HTTP mode when OLLAMA_HOST env is set and agent is "ollama".
func (a *AgentSpec) ExecuteWithContext(ctx context.Context, prompt string) (string, TokenUsage, error) {
	if a.Agent == "ollama" {
		if host := os.Getenv("OLLAMA_HOST"); host != "" {
			return a.executeViaHTTP(ctx, host, prompt)
		}
	}
	return a.executeViaCLI(ctx, prompt)
}

// ─── HTTP Mode (Docker / remote Ollama) ─────────────────────────────────────

// ollamaRequest is the JSON body for POST /api/generate.
type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// ollamaStreamChunk is a single NDJSON line from the streaming response.
type ollamaStreamChunk struct {
	Response        string `json:"response"`
	Done            bool   `json:"done"`
	PromptEvalCount int    `json:"prompt_eval_count,omitempty"`
	EvalCount       int    `json:"eval_count,omitempty"`
}

func (a *AgentSpec) executeViaHTTP(ctx context.Context, host string, prompt string) (string, TokenUsage, error) {
	var usage TokenUsage

	reqBody := ollamaRequest{
		Model:  a.ModelName,
		Prompt: prompt,
		Stream: true,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", usage, fmt.Errorf("failed to marshal ollama request: %v", err)
	}

	url := strings.TrimRight(host, "/") + "/api/generate"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", usage, fmt.Errorf("failed to create HTTP request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 0} // no timeout on client; context handles cancellation
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", usage, fmt.Errorf("ollama HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", usage, fmt.Errorf("ollama returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Stream NDJSON response: print tokens in real-time and accumulate output
	var fullResponse strings.Builder
	decoder := json.NewDecoder(resp.Body)

	for decoder.More() {
		var chunk ollamaStreamChunk
		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return fullResponse.String(), usage, fmt.Errorf("error decoding stream: %v", err)
		}

		// Print token to stdout in real-time (mirrors CLI behavior)
		fmt.Print(chunk.Response)
		fullResponse.WriteString(chunk.Response)

		// The final chunk contains token metrics
		if chunk.Done {
			usage.PromptTokens = chunk.PromptEvalCount
			usage.EvalTokens = chunk.EvalCount
		}
	}
	fmt.Println() // newline after streaming

	return strings.TrimSpace(fullResponse.String()), usage, nil
}

// ─── CLI Mode (local development) ───────────────────────────────────────────

func (a *AgentSpec) executeViaCLI(ctx context.Context, prompt string) (string, TokenUsage, error) {
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

// Compile-time check: ensure unused imports don't break when only one mode is active.
var _ = time.Second
