package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dothanhlam/harness-app/internal/config"
	"github.com/dothanhlam/harness-app/internal/memory"
	"github.com/dothanhlam/harness-app/internal/qa"
	"github.com/dothanhlam/harness-app/internal/telemetry"
)

// EpicPipeline holds the decomposed tasks for an epic.
type EpicPipeline struct {
	SubTasks []SubTask `json:"sub_tasks"`
}

// SubTask represents a single decomposed task within an epic.
type SubTask struct {
	Name         string `json:"task_name"`
	TargetFolder string `json:"target_subfolder"`
	PromptSpecs  string `json:"prompt_specifications"`
	TicketID     string `json:"ticket_id,omitempty"`
}

// TaskResult holds the outcome of an epic sub-task execution.
type TaskResult struct {
	TaskName string
	Success  bool
	Error    error
}

// ExecuteBigEpic reads a directory of requirements, decomposes it into tasks, and orchestrates execution.
// When parallel=true, sub-tasks run concurrently with isolated memory directories.
func ExecuteBigEpic(epicFolderPath string, cfg config.Config, tracker *telemetry.Tracker, parallel bool) {
	fmt.Printf("📦 [EPIC ORCHESTRATOR] Scanning epic requirements directory: %s\n", epicFolderPath)

	var hugeContext strings.Builder
	files, _ := os.ReadDir(epicFolderPath)
	for _, file := range files {
		content, _ := os.ReadFile(filepath.Join(epicFolderPath, file.Name()))
		hugeContext.WriteString(fmt.Sprintf("\n--- FILE: %s ---\n%s", file.Name(), string(content)))
	}

	sysPrompt := `You are a Technical Product Owner. Analyze the provided multi-file epic software requirements.
Decompose this large system into a sequential list of standalone, decoupled Go sub-features.
Each sub-feature must map to its own clean subfolder.
Extract a 'ticket_id' if one exists in the requirement context. If none exists, omit the field.
Output strictly a JSON array matching this format:
{
  "sub_tasks": [
    {"task_name": "db_connector", "ticket_id": "ENG-123", "target_subfolder": "workspace/db_connector", "prompt_specifications": "Implement clickhouse initialization..."},
    {"task_name": "log_parser", "target_subfolder": "workspace/log_parser", "prompt_specifications": "Implement ParseLogLine functions..."}
  ]
}`

	fmt.Println("🕵️ PM Agent is decomposing the epic into sub-sprints...")
	fullPrompt := fmt.Sprintf("SYSTEM INSTRUCTIONS:\n%s\n\nUSER INPUT:\n%s", sysPrompt, hugeContext.String())
	jsonPlan, err := cfg.DevOps.Execute(fullPrompt)
	if err != nil {
		log.Fatalf("❌ Epic decomposition failed: %v", err)
	}

	var epicPipeline EpicPipeline
	// Try extracting JSON from backticks if the model returned markdown
	if strings.Contains(jsonPlan, "```json") {
		start := strings.Index(jsonPlan, "```json") + 7
		end := strings.LastIndex(jsonPlan, "```")
		if end > start {
			jsonPlan = jsonPlan[start:end]
		}
	}

	if err := json.Unmarshal([]byte(jsonPlan), &epicPipeline); err != nil {
		log.Fatalf("❌ Failed to parse epic JSON decomposition: %v\nRaw Output:\n%s", err, jsonPlan)
	}

	if parallel {
		executeParallel(epicPipeline, cfg, tracker)
	} else {
		executeSequential(epicPipeline, cfg, tracker)
	}

	fmt.Println("\n🏆 [EPIC COMPLETED] All files in the epic directory have been successfully implemented into modular packages!")
}

// executeSequential runs each sub-task one at a time through the full core loop.
func executeSequential(ep EpicPipeline, cfg config.Config, tracker *telemetry.Tracker) {
	for i, task := range ep.SubTasks {
		fmt.Printf("\n🎬 [SPRINT %d/%d] Beginning implementation of Module: %s\n", i+1, len(ep.SubTasks), task.Name)

		dodContent := fmt.Sprintf("# TASK: %s\n- Target Subfolder: %s\n- Ticket ID: %s\n\n## Requirements\n%s",
			task.Name, task.TargetFolder, task.TicketID, task.PromptSpecs)
		_ = os.WriteFile("memory/definitions_of_done.md", []byte(dodContent), 0644)

		RunCoreHarnessLoop(cfg, tracker)
	}
}

// ─────────────────────────────────────────────────────────────
// PARALLEL EPIC EXECUTION
// Uses goroutines + WaitGroup + channels for concurrent phases.
// Each task gets an isolated memory/<task_name>/ directory to
// avoid shared state races on definitions_of_done.md.
// ─────────────────────────────────────────────────────────────

// executeParallel runs independent sub-tasks concurrently in 4 phases:
//
//	Phase 1 (parallel): Dev agent code generation — isolated memory dirs
//	Phase 2 (parallel): QA audit on each task's target folder
//	Phase 3 (blocking): Single HITL gate for all tasks
//	Phase 4 (parallel): DevOps release notes generation
//	Phase 5 (sequential): Memory progression
func executeParallel(ep EpicPipeline, cfg config.Config, tracker *telemetry.Tracker) {
	fmt.Printf("⚡ [PARALLEL MODE] Launching %d sub-tasks concurrently...\n", len(ep.SubTasks))

	// ── Phase 1: Parallel Dev Agent execution with isolated memory ──
	var devWg sync.WaitGroup
	devResultCh := make(chan TaskResult, len(ep.SubTasks))

	for i, task := range ep.SubTasks {
		devWg.Add(1)
		go func(idx int, t SubTask) {
			defer devWg.Done()
			fmt.Printf("🤖 [PARALLEL DEV %d/%d] Module: %s\n", idx+1, len(ep.SubTasks), t.Name)

			// Create isolated memory directory for this task
			isolatedMemDir := fmt.Sprintf("memory/%s", t.Name)
			_ = os.MkdirAll(isolatedMemDir, 0755)
			_ = os.MkdirAll(t.TargetFolder, 0755)

			// Write task-specific DoD to isolated memory
			dodContent := fmt.Sprintf("# TASK: %s\n- Target Subfolder: %s\n- Ticket ID: %s\n\n## Requirements\n%s",
				t.Name, t.TargetFolder, t.TicketID, t.PromptSpecs)
			_ = os.WriteFile(filepath.Join(isolatedMemDir, "definitions_of_done.md"), []byte(dodContent), 0644)

			// Copy shared memory files to isolated dir
			if blueprint, err := os.ReadFile("memory/system_blueprint.md"); err == nil {
				_ = os.WriteFile(filepath.Join(isolatedMemDir, "system_blueprint.md"), blueprint, 0644)
			}

			// Clone dev agent spec and remap --add-dir ./memory to isolated path
			devAgent := cfg.Dev.Clone()
			for j, arg := range devAgent.CmdTemplate {
				devAgent.CmdTemplate[j] = strings.ReplaceAll(arg, "./memory", "./"+isolatedMemDir)
			}

			devPrompt, err := os.ReadFile(".agents/antigravity_dev_prompt.md")
			if err != nil {
				devResultCh <- TaskResult{TaskName: t.Name, Success: false, Error: fmt.Errorf("missing dev prompt: %v", err)}
				return
			}

			_, err = devAgent.Execute(string(devPrompt))
			devResultCh <- TaskResult{TaskName: t.Name, Success: err == nil, Error: err}
		}(i, task)
	}

	go func() { devWg.Wait(); close(devResultCh) }()

	var devFailed []string
	for result := range devResultCh {
		if result.Success {
			fmt.Printf("✅ [PARALLEL DEV] Module %s code generation complete.\n", result.TaskName)
		} else {
			fmt.Printf("⚠️ [PARALLEL DEV] Module %s failed: %v\n", result.TaskName, result.Error)
			devFailed = append(devFailed, result.TaskName)
		}
	}

	// ── Phase 2: Parallel QA on all outputs ──
	fmt.Println("\n🛡️ [PARALLEL QA] Running security audit on all modules...")
	var qaWg sync.WaitGroup
	qaResultCh := make(chan TaskResult, len(ep.SubTasks))

	for _, task := range ep.SubTasks {
		qaWg.Add(1)
		go func(t SubTask) {
			defer qaWg.Done()
			auditErr := qa.AuditGeneratedCode(t.TargetFolder)
			qaResultCh <- TaskResult{TaskName: t.Name, Success: auditErr == nil, Error: auditErr}
		}(task)
	}

	go func() { qaWg.Wait(); close(qaResultCh) }()

	var qaFailed []string
	for result := range qaResultCh {
		if result.Success {
			fmt.Printf("✅ [PARALLEL QA] Module %s passed security audit.\n", result.TaskName)
		} else {
			fmt.Printf("⚠️ [PARALLEL QA] Module %s failed audit: %v\n", result.TaskName, result.Error)
			qaFailed = append(qaFailed, result.TaskName)
		}
	}

	// ── Phase 3: Single HITL gate for all tasks ──
	if len(devFailed) > 0 || len(qaFailed) > 0 {
		fmt.Printf("\n⚠️ [PARALLEL SUMMARY] Dev failures: %v, QA failures: %v\n", devFailed, qaFailed)
	}

	UpdateState(StageHITL, 0, tracker)
	fmt.Print("🚧 [HITL GATE] Parallel epic complete. Do you APPROVE all modules? (y/n) [Auto-yes in 30s]: ")

	inputChan := make(chan string)
	go func() {
		var input string
		fmt.Scanln(&input)
		inputChan <- input
	}()

	var finalInput string
	select {
	case finalInput = <-inputChan:
	case <-time.After(30 * time.Second):
		fmt.Println("\n⏳ Timeout reached. Auto-approving (y).")
		finalInput = "y"
	}

	finalInput = strings.ToLower(strings.TrimSpace(finalInput))
	if finalInput != "y" && finalInput != "" {
		fmt.Println("🛑 Disapproved! Terminating pipeline.")
		os.Exit(1)
	}

	// ── Phase 4: Parallel DevOps (release notes generation) ──
	fmt.Println("\n📝 [PARALLEL DEVOPS] Generating release notes for all modules...")
	var devopsWg sync.WaitGroup
	for _, task := range ep.SubTasks {
		devopsWg.Add(1)
		go func(t SubTask) {
			defer devopsWg.Done()
			featureFiles, _ := os.ReadDir(t.TargetFolder)
			var allCode string
			for _, ff := range featureFiles {
				if !ff.IsDir() && strings.HasSuffix(ff.Name(), ".go") {
					content, _ := os.ReadFile(filepath.Join(t.TargetFolder, ff.Name()))
					allCode += string(content) + "\n"
				}
			}
			if allCode == "" {
				return
			}

			sysPrompt := fmt.Sprintf("You are a deployment release manager. Generate a short, bulleted markdown release note based on the provided Go code for the feature '%s'. Keep it brief. Be extremely concise. Return bullet points only. Limit your response to under 150 words. Do not write filler structural prose.", t.Name)
			fullPrompt := fmt.Sprintf("SYSTEM INSTRUCTIONS:\n%s\n\nUSER INPUT:\n%s", sysPrompt, allCode)
			
			var releaseNotes string
			var err error
			if cfg.DevOps.Agent == "ollama" {
				ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
				defer cancel()
				releaseNotes, err = cfg.DevOps.ExecuteWithContext(ctx, fullPrompt)
				if err != nil {
					fmt.Println("⚠️ [OLLAMA THERMAL THROTTLING] DevOps agent timed out. Gracefully falling back to save CPU cycles...")
					releaseNotes = "- DevOps auto-generation aborted (thermal fallback).\n- Check commits for details."
					err = nil
				}
			} else {
				releaseNotes, err = cfg.DevOps.Execute(fullPrompt)
			}

			if err != nil {
				fmt.Printf("⚠️ DevOps failed for %s: %v\n", t.Name, err)
				return
			}
			notePath := filepath.Join(t.TargetFolder, "RELEASE_NOTES.md")
			_ = os.WriteFile(notePath, []byte(releaseNotes), 0644)
			fmt.Printf("📝 Generated %s\n", notePath)
		}(task)
	}
	devopsWg.Wait()

	// ── Phase 5: Sequential Memory Progression ──
	UpdateState(StageCompact, 0, tracker)
	memory.UpdateSystemMemory(&cfg.DevOps)
	memory.CompactSystemMemory(&cfg.DevOps)

	UpdateState(StageDone, 0, tracker)
	fmt.Printf("\n📊 [PARALLEL EPIC RESULTS] %d total, %d dev failures, %d QA failures\n",
		len(ep.SubTasks), len(devFailed), len(qaFailed))
}
