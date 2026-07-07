package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dothanhlam/harness-engineering/internal/config"
	"github.com/dothanhlam/harness-engineering/internal/pipeline"
	"github.com/dothanhlam/harness-engineering/internal/qa"
	"github.com/dothanhlam/harness-engineering/internal/telemetry"
	"encoding/json"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: harness [init|run] [flags]")
		os.Exit(1)
	}

	subcommand := os.Args[1]

	var args []string
	if subcommand != "init" && subcommand != "run" {
		// Fallback to "run" if no subcommand is explicitly passed (backward compat)
		subcommand = "run"
		args = os.Args[1:]
	} else {
		args = os.Args[2:]
	}

	if subcommand == "init" {
		runInit(args)
		return
	} else if subcommand == "run" {
		runPipeline(args)
		return
	}
}

func runInit(args []string) {
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	projectDir := initCmd.String("project-dir", ".", "Target project directory")
	initCmd.Parse(args)

	fmt.Printf("🚀 INITIALIZING HARNESS FRAMEWORK in %s\n", *projectDir)

	harnessDir := filepath.Join(*projectDir, ".harness")
	agentsDir := filepath.Join(*projectDir, ".agents")
	_ = os.MkdirAll(harnessDir, 0755)
	_ = os.MkdirAll(agentsDir, 0755)

	rulesPath := filepath.Join(harnessDir, "rules.json")
	defaultRules := map[string]string{
		"\"os/exec\"":    "invokes forbidden package os/exec",
		"exec.Command":   "invokes forbidden package os/exec",
		"rm -rf":         "contains destructive terminal command 'rm -rf'",
		"os.Remove(":     "contains unauthorized filesystem manipulation",
		"os.RemoveAll(":  "contains unauthorized filesystem manipulation",
		"os.Rename(":     "contains unauthorized filesystem manipulation",
		"password =":     "contains potential hardcoded credentials",
		"secret =":       "contains potential hardcoded credentials",
		"aws_access_key": "contains potential hardcoded credentials",
	}

	data, err := json.MarshalIndent(defaultRules, "", "  ")
	if err == nil {
		_ = os.WriteFile(rulesPath, data, 0644)
	}

	promptPath := filepath.Join(agentsDir, "antigravity_dev_prompt.md")
	promptContent := "You are an expert Developer agent.\nWhen modifying existing files, output SEARCH/REPLACE blocks formatted exactly like this:\n### FILE: /path/to/file.go\n<<<<\nold code to replace\n====\nnew code\n>>>>\n"
	_ = os.WriteFile(promptPath, []byte(promptContent), 0644)

	fmt.Println("✅ Initialization complete. Run 'harness run' to start the pipeline.")
}

func runPipeline(args []string) {
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)

	// CLI flags
	projectDir := runCmd.String("project-dir", ".", "Target project directory")
	taskFlag := runCmd.String("task", "", "Raw requirement to trigger Phase 0 Business Analyst")
	epicFlag := runCmd.String("epic", "", "Path to a directory containing epic requirements")
	targetFlag := runCmd.String("target", "", "Explicit target subfolder")
	forceRegression := runCmd.Bool("force-regression", false, "Force QA regression tests")
	parallelEpic := runCmd.Bool("parallel-epic", false, "Run epic sub-tasks concurrently")
	
	baAgentCmd := runCmd.String("ba-agent", "", "Command/binary to execute for Phase 0")
	baModelCmd := runCmd.String("ba-model", "", "Model name for the Phase 0 agent")
	devAgentCmd := runCmd.String("dev-agent", "", "Command/binary to execute for Phase 1")
	devAgentModel := runCmd.String("dev-model", "", "Model name for the Dev agent")
	devOpsAgent := runCmd.String("devops-agent", "", "CLI agent to execute for Phase 3")
	devOpsModel := runCmd.String("devops-model", "", "Model name for Phase 3")
	
	runCmd.Parse(args)

	// Change working directory to ProjectDir so relative paths work automatically
	if *projectDir != "." {
		err := os.Chdir(*projectDir)
		if err != nil {
			log.Fatalf("❌ Failed to change directory to %s: %v", *projectDir, err)
		}
	}

	startTime := time.Now()
	tracker := telemetry.NewTracker(startTime)

	// Load configuration
	cfg := config.LoadConfig("harness_config.json")

	// Apply CLI overrides to config
	cfg.ProjectDir = *projectDir
	cfg.ForceRegression = *forceRegression
	if *baAgentCmd != "" { cfg.BA.Agent = *baAgentCmd }
	if *baModelCmd != "" { cfg.BA.ModelName = *baModelCmd }
	if *devAgentCmd != "" { cfg.Dev.Agent = *devAgentCmd }
	if *devAgentModel != "" { cfg.Dev.ModelName = *devAgentModel }
	if *devOpsAgent != "" { cfg.DevOps.Agent = *devOpsAgent }
	if *devOpsModel != "" { cfg.DevOps.ModelName = *devOpsModel }

	// Try loading custom QA Rules if present
	if rulesData, err := os.ReadFile(".harness/rules.json"); err == nil {
		var customRules map[string]string
		if json.Unmarshal(rulesData, &customRules) == nil {
			cfg.QARules = customRules
		}
	}

	fmt.Printf("🚀 ACTIVATING HARNESS PIPELINE in %s\n", cfg.ProjectDir)
	_ = os.MkdirAll("memory", 0755)
	
	// Create workspace if it doesn't exist (legacy fallback)
	_ = os.MkdirAll("workspace", 0755)

	fmt.Printf("⚙️  Configuration:\n")
	fmt.Printf("   - BA Agent:    %s\n", cfg.BA.Agent)
	fmt.Printf("   - Dev Agent:   %s (model: %s)\n", cfg.Dev.Agent, cfg.Dev.ModelName)
	fmt.Printf("   - DevOps Agent: %s\n", cfg.DevOps.Agent)
	fmt.Printf("   - DevOps Model: %s\n", cfg.DevOps.ModelName)

	if *epicFlag != "" {
		pipeline.ExecuteBigEpic(*epicFlag, cfg, tracker, *parallelEpic)
		linesGenerated := qa.CountGeneratedLines(".")
		_ = tracker.Finalize("telemetry.json", linesGenerated)
		return
	}

	if *taskFlag != "" {
		fmt.Printf("\n🎯 Raw requirement received: '%s'\n", *taskFlag)
		fmt.Printf("🤖 BA Agent (%s) is drafting the Definitions of Done...\n", cfg.BA.Agent)

		var targetInstructions string
		if *targetFlag != "" {
			targetInstructions = fmt.Sprintf("Start the output with exactly: \"# TASK: <snake_case_name>\"\nFollowed by: \"- Target Subfolder: %s\"", *targetFlag)
		} else {
			targetInstructions = "Start the output with exactly: \"# TASK: <snake_case_name>\"\nFollowed by: \"- Target Subfolder: workspace/<snake_case_name>\""
		}

		baPrompt := fmt.Sprintf(`
You are an expert Business Analyst. 
Take this raw requirement: "%s".
Analyze it and generate a standardized, highly technical 'definitions_of_done.md' layout.
%s
Then, output the strict markdown checklist content. Do not include any chat filler or explanations.
`, *taskFlag, targetInstructions)

		outBA, tu, err := cfg.BA.Execute(baPrompt)
		tracker.AddTokens(tu.PromptTokens, tu.EvalTokens)
		if err != nil {
			log.Fatalf("❌ BA Agent failed: %v", err)
		}

		_ = os.WriteFile("memory/definitions_of_done.md", []byte(outBA), 0644)
		fmt.Println("✅ Successfully generated memory/definitions_of_done.md.")
	}

	pipeline.RunCoreHarnessLoop(cfg, tracker)

	linesGenerated := qa.CountGeneratedLines(".")
	_ = tracker.Finalize("telemetry.json", linesGenerated)
}
