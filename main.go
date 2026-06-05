package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dothanhlam/harness-app/internal/config"
	"github.com/dothanhlam/harness-app/internal/pipeline"
	"github.com/dothanhlam/harness-app/internal/qa"
	"github.com/dothanhlam/harness-app/internal/telemetry"
)

func main() {
	startTime := time.Now()
	tracker := telemetry.NewTracker(startTime)

	// Load configuration from harness_config.json (merges with defaults)
	cfg := config.LoadConfig("harness_config.json")

	// CLI flags with config-based defaults
	taskFlag := flag.String("task", "", "Raw requirement to trigger Phase 0 Business Analyst")
	epicFlag := flag.String("epic", "", "Path to a directory containing epic requirements for decomposition")
	parallelEpic := flag.Bool("parallel-epic", false, "Run epic sub-tasks concurrently with isolated workspaces")
	baAgentCmd := flag.String("ba-agent", cfg.BA.Agent, "Command/binary to execute for Phase 0 Business Analyst")
	baModelCmd := flag.String("ba-model", cfg.BA.ModelName, "Model name for the Phase 0 Business Analyst agent")
	devAgentCmd := flag.String("dev-agent", cfg.Dev.Agent, "Command/binary to execute for Phase 1 Developer Coding")
	devAgentModel := flag.String("dev-model", cfg.Dev.ModelName, "Model name for the Dev agent (sets ANTIGRAVITY_MODEL env var)")
	devOpsAgent := flag.String("devops-agent", cfg.DevOps.Agent, "CLI agent to execute for Phase 3 DevOps documentation (e.g., ollama)")
	devOpsModel := flag.String("devops-model", cfg.DevOps.ModelName, "Model name to execute for Phase 3 DevOps documentation")
	flag.Parse()

	// Apply CLI overrides to config
	cfg.BA.Agent = *baAgentCmd
	cfg.BA.ModelName = *baModelCmd
	cfg.Dev.Agent = *devAgentCmd
	cfg.Dev.ModelName = *devAgentModel
	cfg.DevOps.Agent = *devOpsAgent
	cfg.DevOps.ModelName = *devOpsModel

	fmt.Println("🚀 ACTIVATING GO HARNESS PIPELINE v2026.1")
	_ = os.MkdirAll("memory", 0755)
	_ = os.MkdirAll("workspace", 0755)

	fmt.Printf("⚙️  Configuration:\n")
	fmt.Printf("   - BA Agent:    %s\n", cfg.BA.Agent)
	fmt.Printf("   - Dev Agent:   %s (model: %s)\n", cfg.Dev.Agent, cfg.Dev.ModelName)
	fmt.Printf("   - DevOps Agent: %s\n", cfg.DevOps.Agent)
	fmt.Printf("   - DevOps Model: %s\n", cfg.DevOps.ModelName)

	// Dispatch: Epic mode or single-task mode
	if *epicFlag != "" {
		pipeline.ExecuteBigEpic(*epicFlag, cfg, tracker, *parallelEpic)
		linesGenerated := qa.CountGeneratedLines("workspace")
		_ = tracker.Finalize("workspace/telemetry.json", linesGenerated)
		return
	}

	// Phase 0: BA Agent generates Definitions of Done from raw requirement
	if *taskFlag != "" {
		fmt.Printf("\n🎯 Raw requirement received: '%s'\n", *taskFlag)
		fmt.Printf("🤖 BA Agent (%s) is drafting the Definitions of Done...\n", cfg.BA.Agent)

		baPrompt := fmt.Sprintf(`
You are an expert Business Analyst. 
Take this raw requirement: "%s".
Analyze it and generate a standardized, highly technical 'definitions_of_done.md' layout.
Output ONLY the strict markdown checklist content. Do not include any chat filler or explanations.
`, *taskFlag)

		outBA, tu, err := cfg.BA.Execute(baPrompt)
		tracker.AddTokens(tu.PromptTokens, tu.EvalTokens)
		if err != nil {
			log.Fatalf("❌ BA Agent failed: %v", err)
		}

		_ = os.WriteFile("memory/definitions_of_done.md", []byte(outBA), 0644)
		fmt.Println("✅ Successfully generated memory/definitions_of_done.md.")
	}

	// Run the core pipeline loop
	pipeline.RunCoreHarnessLoop(cfg, tracker)

	// Finalize telemetry
	linesGenerated := qa.CountGeneratedLines("workspace")
	_ = tracker.Finalize("workspace/telemetry.json", linesGenerated)
}
