package config

import (
	"encoding/json"
	"os"

	"github.com/dothanhlam/harness-app/internal/agent"
)

// Config holds the full harness pipeline configuration with pluggable agent specs.
type Config struct {
	BA     agent.AgentSpec `json:"ba"`
	Dev    agent.AgentSpec `json:"dev"`
	DevOps agent.AgentSpec `json:"devops"`
	Mem0   Mem0Config      `json:"mem0"`
}

// Mem0Config holds the configuration for the Mem0 REST client.
type Mem0Config struct {
	BaseURL string `json:"base_url"`
	AgentID string `json:"agent_id"`
}

// DefaultConfig returns the built-in default configuration.
func DefaultConfig() Config {
	return Config{
		BA: agent.AgentSpec{
			Agent:       "gemini",
			CmdTemplate: []string{"run", "{prompt}"},
			MCPConfig:   ".mcp/ba_notion.json",
		},
		Dev: agent.AgentSpec{
			Agent:       "agy",
			ModelName:   "gemini-2.5-flash",
			CmdTemplate: []string{"--print", "{prompt}", "--dangerously-skip-permissions", "--add-dir", "./workspace", "--add-dir", "./memory"},
			Env:         map[string]string{"ANTIGRAVITY_MODEL": "{model}"},
		},
		DevOps: agent.AgentSpec{
			Agent:       "ollama",
			ModelName:   "hermes3:8b",
			CmdTemplate: []string{"run", "{model}", "{prompt}"},
			MCPConfig:   ".mcp/devops_linear.json",
		},
		Mem0: Mem0Config{
			BaseURL: "http://localhost:8000",
			AgentID: "harness-architect",
		},
	}
}

// LoadConfig reads harness_config.json from the given path and merges with defaults.
// If the file doesn't exist or is invalid, defaults are returned.
func LoadConfig(path string) Config {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, &cfg)
	}
	return cfg
}
