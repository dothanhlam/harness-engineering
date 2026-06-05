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
}

// DefaultConfig returns the built-in default configuration.
func DefaultConfig() Config {
	return Config{
		BA: agent.AgentSpec{
			Agent:       "ollama",
			ModelName:   "hermes3:8b",
			CmdTemplate: []string{"run", "{model}", "{prompt}", "--verbose"},
		},
		Dev: agent.AgentSpec{
			Agent:       "ollama",
			ModelName:   "gemma4:e4b",
			CmdTemplate: []string{"run", "{model}", "{prompt}", "--verbose"},
		},
		DevOps: agent.AgentSpec{
			Agent:       "ollama",
			ModelName:   "hermes3:8b",
			CmdTemplate: []string{"run", "{model}", "{prompt}", "--verbose"},
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
