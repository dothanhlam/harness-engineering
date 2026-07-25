package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dothanhlam/harness-engineering/internal/config"
	"github.com/dothanhlam/harness-engineering/internal/events"
	"github.com/dothanhlam/harness-engineering/internal/monitor"
	"github.com/dothanhlam/harness-engineering/internal/pipeline"
	"github.com/dothanhlam/harness-engineering/internal/qa"
	"github.com/dothanhlam/harness-engineering/internal/telemetry"
	"encoding/json"
	"path/filepath"
)

// eventLogPath is the append-only stream a monitor process tails. It lives in
// workspace/ beside state.json, and is truncated at the start of every run.
const eventLogPath = "workspace/events.jsonl"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: harness [init|run|monitor] [flags]")
		os.Exit(1)
	}

	subcommand := os.Args[1]

	var args []string
	switch subcommand {
	case "init", "run", "monitor":
		args = os.Args[2:]
	default:
		// Fallback to "run" if no subcommand is explicitly passed (backward compat)
		subcommand = "run"
		args = os.Args[1:]
	}

	switch subcommand {
	case "init":
		runInit(args)
	case "run":
		runPipeline(args)
	case "monitor":
		runMonitor(args)
	}
}

// runMonitor tails the event stream of a run in progress (or replays a finished
// one) and renders it to the terminal. It is a separate process from the run:
// the run stays headless and owns stdin for the HITL gate.
func runMonitor(args []string) {
	monCmd := flag.NewFlagSet("monitor", flag.ExitOnError)
	projectDir := monCmd.String("project-dir", ".", "Project directory whose run to monitor")
	path := monCmd.String("path", "", "Event log path (default: <project-dir>/workspace/events.jsonl)")
	once := monCmd.Bool("once", false, "Replay the current log and exit instead of following live")
	noOutput := monCmd.Bool("no-output", false, "Hide mirrored agent output for a high-level view")
	color := monCmd.String("color", "auto", "Colorize output: auto|always|never")
	web := monCmd.Bool("web", false, "Serve a browser dashboard instead of tailing the terminal")
	port := monCmd.Int("port", 7777, "Port for --web (bound to localhost only)")
	monCmd.Parse(args)

	logPath := *path
	if logPath == "" {
		logPath = filepath.Join(*projectDir, eventLogPath)
	}

	// Ctrl-C ends the session cleanly rather than killing mid-write.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *web {
		addr := fmt.Sprintf("127.0.0.1:%d", *port)
		fmt.Printf("🌐 Monitor dashboard at http://%s  (Ctrl-C to stop)\n", addr)
		if err := monitor.Serve(ctx, addr, logPath); err != nil {
			log.Fatalf("❌ Monitor server error: %v", err)
		}
		return
	}

	opts := monitor.Options{
		Once:       *once,
		ShowOutput: !*noOutput,
	}
	switch *color {
	case "always":
		on := true
		opts.Color = &on
	case "never":
		off := false
		opts.Color = &off
	}

	if !*once {
		fmt.Printf("👁️  Monitoring %s (Ctrl-C to stop)\n", logPath)
	}
	if err := monitor.Follow(ctx, logPath, os.Stdout, opts); err != nil {
		log.Fatalf("❌ Monitor error: %v", err)
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

	// Open the event stream. A failure here is non-fatal: monitoring must never
	// be able to take the pipeline down, and a nil bus is a no-op everywhere.
	//
	// With neither flag set the run resumes whatever memory/definitions_of_done.md
	// already holds, so the label comes from that DoD rather than the empty flag.
	mode, label := "task", *taskFlag
	switch {
	case *epicFlag != "":
		mode, label = "epic", filepath.Base(strings.TrimRight(*epicFlag, "/"))
	case *taskFlag == "":
		dod, _ := os.ReadFile("memory/definitions_of_done.md")
		mode, label = "resume", pipeline.FeatureNameFromDoD(string(dod))
	}
	bus, err := events.Open(eventLogPath, events.NewRunID(label, startTime))
	if err != nil {
		fmt.Printf("⚠️ Event log unavailable (%v); continuing without monitoring.\n", err)
	}
	defer bus.Close()

	bus.Emit(events.KindRunStarted, "", events.RunStarted{
		Mode:       mode,
		Task:       *taskFlag,
		Epic:       *epicFlag,
		ProjectDir: cfg.ProjectDir,
		Parallel:   *parallelEpic,
		MaxRetries: pipeline.MaxRetries,
		Agents: map[string]string{
			events.RoleBA:     agentLabel(cfg.BA.Agent, cfg.BA.ModelName),
			events.RoleDev:    agentLabel(cfg.Dev.Agent, cfg.Dev.ModelName),
			events.RoleDevOps: agentLabel(cfg.DevOps.Agent, cfg.DevOps.ModelName),
		},
	})

	fmt.Printf("⚙️  Configuration:\n")
	fmt.Printf("   - BA Agent:    %s\n", cfg.BA.Agent)
	fmt.Printf("   - Dev Agent:   %s (model: %s)\n", cfg.Dev.Agent, cfg.Dev.ModelName)
	fmt.Printf("   - DevOps Agent: %s\n", cfg.DevOps.Agent)
	fmt.Printf("   - DevOps Model: %s\n", cfg.DevOps.ModelName)
	if bus != nil {
		fmt.Printf("   - Event log:   %s (run %s)\n", eventLogPath, bus.RunID())
	}

	if *epicFlag != "" {
		failed := pipeline.ExecuteBigEpic(*epicFlag, cfg, tracker, *parallelEpic, bus)
		linesGenerated := qa.CountGeneratedLines(".")
		_ = tracker.Finalize("telemetry.json", linesGenerated)

		// A parallel epic reports failures and completes rather than aborting,
		// so the terminal event must not claim success when modules failed.
		reason := ""
		if len(failed) > 0 {
			reason = fmt.Sprintf("%d module(s) failed: %s", len(failed), strings.Join(failed, ", "))
		}
		events.EmitRunFinished(bus, "", tracker, len(failed) == 0, reason, failed...)
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

		outBA, tu, err := events.Run(bus, "", events.RoleBA, &cfg.BA, baPrompt)
		tracker.AddTokens(tu.PromptTokens, tu.EvalTokens)
		if err != nil {
			events.EmitRunFinished(bus, "", tracker, false, fmt.Sprintf("BA agent failed: %v", err))
			log.Fatalf("❌ BA Agent failed: %v", err)
		}

		_ = os.WriteFile("memory/definitions_of_done.md", []byte(outBA), 0644)
		fmt.Println("✅ Successfully generated memory/definitions_of_done.md.")
	}

	pipeline.RunCoreHarnessLoop(cfg, tracker, bus)

	linesGenerated := qa.CountGeneratedLines(".")
	_ = tracker.Finalize("telemetry.json", linesGenerated)
	events.EmitRunFinished(bus, "", tracker, true, "")
}

// agentLabel renders an agent and its model for display, e.g. "llama_cpp (hermes3_8b)".
func agentLabel(name, model string) string {
	if model == "" {
		return name
	}
	return fmt.Sprintf("%s (%s)", name, model)
}
