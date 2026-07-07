package config

import (
	"encoding/json"
	"os"

	"github.com/dothanhlam/harness-engineering/internal/agent"
)

// Config holds the full harness pipeline configuration with pluggable agent specs.
type Config struct {
	ContextWindow   int             `json:"context_window"`
	FlashAttention  bool            `json:"flash_attention"`
	ProjectDir      string          `json:"project_dir"`
	ForceRegression bool            `json:"force_regression"`
	BA              agent.AgentSpec `json:"ba"`
	Dev            agent.AgentSpec `json:"dev"`
	DevOps         agent.AgentSpec   `json:"devops"`
	QAIgnore       []string          `json:"qa_ignore"`
	QARules        map[string]string `json:"qa_rules"`
}

// DefaultConfig returns the built-in default configuration.
func DefaultConfig() Config {
	return Config{
		ContextWindow:  16384,
		FlashAttention: true,
		BA: agent.AgentSpec{
			Agent:     "llama_cpp",
			ModelName: "~/ai_models/hermes3_8b.gguf",
		},
		Dev: agent.AgentSpec{
			Agent:     "llama_cpp",
			ModelName: "~/ai_models/gemma4_e4b.gguf",
		},
		DevOps: agent.AgentSpec{
			Agent:     "llama_cpp",
			ModelName: "~/ai_models/hermes3_8b.gguf",
		},
		QAIgnore: []string{},
		QARules: map[string]string{
			"\"os/exec\"":    "invokes forbidden package os/exec",
			"exec.Command":   "invokes forbidden package os/exec",
			"rm -rf":         "contains destructive terminal command 'rm -rf'",
			"os.Remove(":     "contains unauthorized filesystem manipulation",
			"os.RemoveAll(":  "contains unauthorized filesystem manipulation",
			"os.Rename(":     "contains unauthorized filesystem manipulation",
			"password =":     "contains potential hardcoded credentials",
			"secret =":       "contains potential hardcoded credentials",
			"aws_access_key": "contains potential hardcoded credentials",
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
	cfg.BA.ContextWindow = cfg.ContextWindow
	cfg.BA.FlashAttention = cfg.FlashAttention
	cfg.Dev.ContextWindow = cfg.ContextWindow
	cfg.Dev.FlashAttention = cfg.FlashAttention
	cfg.DevOps.ContextWindow = cfg.ContextWindow
	cfg.DevOps.FlashAttention = cfg.FlashAttention
	return cfg
}
