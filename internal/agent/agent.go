package agent

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// AgentSpec defines a pluggable CLI agent with dynamic command templates and environment injection.
type AgentSpec struct {
	Agent       string            `json:"agent"`
	ModelName   string            `json:"model_name,omitempty"`
	CmdTemplate []string          `json:"cmd_template"`
	MCPConfig   string            `json:"mcp_config,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
}

// Execute runs the agent CLI with the given prompt, replacing template tokens dynamically.
// Supported tokens: {prompt}, {model}, {mcp}
func (a *AgentSpec) Execute(prompt string) (string, error) {
	if len(a.CmdTemplate) == 0 {
		return "", fmt.Errorf("agent %s has no cmd_template defined", a.Agent)
	}

	var finalArgs []string
	for _, arg := range a.CmdTemplate {
		arg = strings.ReplaceAll(arg, "{prompt}", prompt)
		arg = strings.ReplaceAll(arg, "{model}", a.ModelName)
		arg = strings.ReplaceAll(arg, "{mcp}", a.MCPConfig)
		finalArgs = append(finalArgs, arg)
	}

	cmd := exec.Command(a.Agent, finalArgs...)

	if len(a.Env) > 0 {
		envVars := os.Environ()
		for k, v := range a.Env {
			v = strings.ReplaceAll(v, "{prompt}", prompt)
			v = strings.ReplaceAll(v, "{model}", a.ModelName)
			v = strings.ReplaceAll(v, "{mcp}", a.MCPConfig)
			envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = envVars
	}

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s CLI error: %v, stderr: %s", a.Agent, err, stderr.String())
	}

	return strings.TrimSpace(out.String()), nil
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
