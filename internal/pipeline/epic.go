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

	"github.com/dothanhlam/harness-engineering/internal/agent"
	"github.com/dothanhlam/harness-engineering/internal/config"
	"github.com/dothanhlam/harness-engineering/internal/events"
	"github.com/dothanhlam/harness-engineering/internal/memory"
	"github.com/dothanhlam/harness-engineering/internal/qa"
	"github.com/dothanhlam/harness-engineering/internal/telemetry"
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
// bus may be nil, in which case no events are emitted.
//
// It returns the names of sub-tasks that failed. Only the parallel path can
// return a non-empty slice: it reports failures and carries on, whereas the
// sequential path aborts the process on the first failure. Callers must treat a
// non-empty result as a failed run.
func ExecuteBigEpic(epicFolderPath string, cfg config.Config, tracker *telemetry.Tracker, parallel bool, bus *events.Bus) []string {
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
	jsonPlan, doUsage, err := events.Run(bus, "", events.RoleDevOps, &cfg.DevOps, fullPrompt)
	tracker.AddTokens(doUsage.PromptTokens, doUsage.EvalTokens)
	if err != nil {
		events.EmitRunFinished(bus, "", tracker, false, fmt.Sprintf("epic decomposition failed: %v", err))
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
		events.EmitRunFinished(bus, "", tracker, false, fmt.Sprintf("epic JSON decomposition unparseable: %v", err))
		log.Fatalf("❌ Failed to parse epic JSON decomposition: %v\nRaw Output:\n%s", err, jsonPlan)
	}

	var failed []string
	if parallel {
		failed = executeParallel(epicPipeline, cfg, tracker, bus)
	} else {
		executeSequential(epicPipeline, cfg, tracker, bus)
	}

	if len(failed) > 0 {
		fmt.Printf("\n⚠️ [EPIC COMPLETED WITH FAILURES] %d module(s) failed: %s\n", len(failed), strings.Join(failed, ", "))
		return failed
	}

	fmt.Println("\n🏆 [EPIC COMPLETED] All files in the epic directory have been successfully implemented into modular packages!")
	return nil
}

// executeSequential runs each sub-task one at a time through the full core loop.
func executeSequential(ep EpicPipeline, cfg config.Config, tracker *telemetry.Tracker, bus *events.Bus) {
	for i, task := range ep.SubTasks {
		fmt.Printf("\n🎬 [SPRINT %d/%d] Beginning implementation of Module: %s\n", i+1, len(ep.SubTasks), task.Name)

		dodContent := fmt.Sprintf("# TASK: %s\n- Target Subfolder: %s\n- Ticket ID: %s\n\n## Requirements\n%s",
			task.Name, task.TargetFolder, task.TicketID, task.PromptSpecs)
		_ = os.WriteFile("memory/definitions_of_done.md", []byte(dodContent), 0644)

		RunCoreHarnessLoop(cfg, tracker, bus)
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
// It returns the names of every sub-task that failed dev or QA.
func executeParallel(ep EpicPipeline, cfg config.Config, tracker *telemetry.Tracker, bus *events.Bus) []string {
	fmt.Printf("⚡ [PARALLEL MODE] Launching %d sub-tasks concurrently...\n", len(ep.SubTasks))

	// ── Phase 1: Parallel Dev Agent execution with isolated memory ──
	var devWg sync.WaitGroup
	devResultCh := make(chan TaskResult, len(ep.SubTasks))

	// A local llama.cpp model is memory-bound, not CPU-bound: each concurrent
	// agent loads its own full copy of the weights, so N sub-tasks means N times
	// the resident set and a host that swaps itself to death. Serialize the model
	// while leaving each task's setup and file writing concurrent. CLI agents
	// (Claude et al.) are network-bound and still fan out freely.
	devSlots := len(ep.SubTasks)
	if cfg.Dev.Agent == "llama_cpp" {
		devSlots = 1
		fmt.Println("🧠 [MEMORY GUARD] Local model detected — dev agents run one at a time.")
	}
	if devSlots < 1 {
		devSlots = 1
	}
	devSem := make(chan struct{}, devSlots)

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

			// Clone dev agent spec and remap --add-dir ./memory to isolated path (CLI agents)
			devAgent := cfg.Dev.Clone()
			for j, arg := range devAgent.CmdTemplate {
				devAgent.CmdTemplate[j] = strings.ReplaceAll(arg, "./memory", "./"+isolatedMemDir)
			}

			// Bus-only: UpdateState also writes workspace/state.json, and these
			// goroutines run concurrently — they would race on that file.
			emitStage(bus, StageDev, t.Name, 0)

			devPrompt, err := os.ReadFile(".agents/antigravity_dev_prompt.md")
			if err != nil {
				devResultCh <- TaskResult{TaskName: t.Name, Success: false, Error: fmt.Errorf("missing dev prompt: %v", err)}
				return
			}

			// Local llama.cpp models have no filesystem tools: inject context inline.
			var finalPrompt string
			if devAgent.Agent == "llama_cpp" {
				blueprint, _ := os.ReadFile("memory/system_blueprint.md")
				finalPrompt = buildLlamaDevPrompt(string(devPrompt), t.Name, dodContent, string(blueprint), "")
			} else {
				finalPrompt = string(devPrompt)
			}

			// Hold a slot only for the model invocation itself.
			devSem <- struct{}{}
			var out string
			var devUsage agent.TokenUsage
			if devAgent.Agent == "llama_cpp" {
				ctx, cancel := context.WithTimeout(context.Background(), devTimeout)
				out, devUsage, err = events.RunWithContext(ctx, bus, t.Name, events.RoleDev, &devAgent, finalPrompt)
				if ctx.Err() == context.DeadlineExceeded {
					err = fmt.Errorf("dev agent exceeded %s — the model may not fit in memory; try a smaller model or a lower context_window", devTimeout)
				}
				cancel()
			} else {
				out, devUsage, err = events.Run(bus, t.Name, events.RoleDev, &devAgent, finalPrompt)
			}
			<-devSem
			tracker.AddTokens(devUsage.PromptTokens, devUsage.EvalTokens)

			// Write out the generated files (llama.cpp emits code as text).
			if devAgent.Agent == "llama_cpp" {
				parseAndWriteGeneratedFiles(out, t.TargetFolder, t.Name, bus)
			}
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

	// ── Phase 2: Parallel QA on all outputs (security audit + test suite) ──
	fmt.Println("\n🛡️ [PARALLEL QA] Running security audit & test suite on all modules...")
	var qaWg sync.WaitGroup
	qaResultCh := make(chan TaskResult, len(ep.SubTasks))

	for _, task := range ep.SubTasks {
		qaWg.Add(1)
		go func(t SubTask) {
			defer qaWg.Done()
			emitStage(bus, StageQA, t.Name, 0)

			var qaErr error
			result := events.QAResult{Target: t.TargetFolder, Attempt: 1, MaxRetries: 1}
			if auditErr := qa.AuditGeneratedCode(t.TargetFolder, cfg.QAIgnore, cfg.QARules); auditErr != nil {
				qaErr = fmt.Errorf("security audit: %v", auditErr)
				result.AuditError = events.TruncateText(auditErr.Error(), events.MaxErrorText)
			} else if testResult := qa.RunTests(t.TargetFolder, false, nil); testResult.Err != nil {
				qaErr = fmt.Errorf("tests failed:\n%s", string(testResult.Output))
				result.TestError = events.TruncateText(string(testResult.Output), events.MaxErrorText)
			}
			result.Passed = qaErr == nil
			bus.Emit(events.KindQAResult, t.Name, result)

			qaResultCh <- TaskResult{TaskName: t.Name, Success: qaErr == nil, Error: qaErr}
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

	UpdateState(StageHITL, "", 0, tracker, bus)
	bus.Emit(events.KindHITLPrompt, "", events.HITLPrompt{TimeoutSeconds: int(hitlTimeout / time.Second)})
	fmt.Printf("🚧 [HITL GATE] Parallel epic complete. Do you APPROVE all modules? (y/n) [Auto-yes in %s]: ", hitlTimeout)

	inputChan := make(chan string)
	go func() {
		var input string
		fmt.Scanln(&input)
		inputChan <- input
	}()

	var finalInput string
	auto := false
	select {
	case finalInput = <-inputChan:
	case <-time.After(hitlTimeout):
		fmt.Println("\n⏳ Timeout reached. Auto-approving (y).")
		finalInput = "y"
		auto = true
	}

	finalInput = strings.ToLower(strings.TrimSpace(finalInput))
	approved := finalInput == "y" || finalInput == ""
	bus.Emit(events.KindHITLResolved, "", events.HITLResolved{Approved: approved, Auto: auto})
	if !approved {
		fmt.Println("🛑 Disapproved! Terminating pipeline.")
		events.EmitRunFinished(bus, "", tracker, false, "rejected at HITL gate")
		os.Exit(1)
	}

	// ── Phase 4: Parallel DevOps (release notes generation) ──
	fmt.Println("\n📝 [PARALLEL DEVOPS] Generating release notes for all modules...")
	var devopsWg sync.WaitGroup
	for _, task := range ep.SubTasks {
		devopsWg.Add(1)
		go func(t SubTask) {
			defer devopsWg.Done()
			emitStage(bus, StageDevOps, t.Name, 0)
			generateReleaseNotes(&cfg.DevOps, t.TargetFolder, t.Name, tracker, bus)
		}(task)
	}
	devopsWg.Wait()

	// ── Phase 5: Sequential Memory Progression ──
	// Run-scoped, not per-task: this folds every module into one blueprint.
	UpdateState(StageCompact, "", 0, tracker, bus)
	memory.UpdateSystemMemory(&cfg.DevOps, tracker, bus, "")
	memory.CompactSystemMemory(&cfg.DevOps, tracker, bus, "")

	UpdateState(StageDone, "", 0, tracker, bus)
	fmt.Printf("\n📊 [PARALLEL EPIC RESULTS] %d total, %d dev failures, %d QA failures\n",
		len(ep.SubTasks), len(devFailed), len(qaFailed))

	// A module can fail dev and then QA; report each name once.
	seen := make(map[string]bool, len(devFailed)+len(qaFailed))
	var failed []string
	for _, name := range append(append([]string{}, devFailed...), qaFailed...) {
		if !seen[name] {
			seen[name] = true
			failed = append(failed, name)
		}
	}
	return failed
}
