package pipeline

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dothanhlam/harness-engineering/internal/agent"
	"github.com/dothanhlam/harness-engineering/internal/config"
	"github.com/dothanhlam/harness-engineering/internal/docs"
	"github.com/dothanhlam/harness-engineering/internal/events"
	"github.com/dothanhlam/harness-engineering/internal/memory"
	"github.com/dothanhlam/harness-engineering/internal/qa"
	"github.com/dothanhlam/harness-engineering/internal/telemetry"
)

// RunCoreHarnessLoop runs the core Development → QA → HITL → DevOps → Memory loop.
// Uses goroutines for parallel QA (audit ∥ tests).
// Memory progression is fully sequential per design decision.
// bus may be nil, in which case no events are emitted.
func RunCoreHarnessLoop(cfg config.Config, tracker *telemetry.Tracker, bus *events.Bus) {
	// =======================================================
	// CORE LOOP: DEV 🔀 QA (SELF-HEALING) & DELEGATION
	// =======================================================
	_ = os.Remove("workspace/qa_error.log")
	_ = os.RemoveAll("workspace/default_task")
	success := false
	maxDelegations := 1

	// Context Reset: ensure the Dev agent starts with a clear loop
	_ = os.RemoveAll(".antigravitycli")
	_ = os.RemoveAll(".claude")

	// Ignore existing directories so we only test newly generated code
	if existingDirs, err := os.ReadDir("workspace"); err == nil {
		for _, d := range existingDirs {
			if d.IsDir() {
				cfg.QAIgnore = append(cfg.QAIgnore, d.Name())
			}
		}
	}

	for delegation := 0; delegation <= maxDelegations; delegation++ {
		for retry := 0; retry < MaxRetries; retry++ {
			// ── PHASE 1: DEVELOPMENT / REPAIR ──
			// The DoD is re-read every attempt: a delegation cycle rewrites it.
			// It is parsed before the state transition so events carry the task.
			dod, _ := os.ReadFile("memory/definitions_of_done.md")
			targetSubfolder, parsedFeatureName := parseDoDTarget(string(dod))

			UpdateState(StageDev, parsedFeatureName, retry, tracker, bus)
			fmt.Printf("🤖 Activating %s CLI for Code Generation/Repair...\n", cfg.Dev.Agent)

			devPrompt, err := os.ReadFile(".agents/antigravity_dev_prompt.md")
			if err != nil {
				events.EmitRunFinished(bus, parsedFeatureName, tracker, false,
					"missing .agents/antigravity_dev_prompt.md")
				log.Fatalf("❌ Missing configuration file: .agents/antigravity_dev_prompt.md")
			}

			// Context Injection for local LLMs (they lack filesystem read tools)
			var finalPrompt string
			if cfg.Dev.Agent == "llama_cpp" {
				blueprint, _ := os.ReadFile("memory/system_blueprint.md")
				qaErr, _ := os.ReadFile("workspace/qa_error.log")
				finalPrompt = buildLlamaDevPrompt(string(devPrompt), parsedFeatureName, string(dod), string(blueprint), string(qaErr))
			} else {
				finalPrompt = string(devPrompt)
			}

			// llama.cpp runs are bounded: an oversized local model wedges rather
			// than failing, which would otherwise hang the run forever.
			var outDev string
			var devUsage agent.TokenUsage
			if cfg.Dev.Agent == "llama_cpp" {
				ctx, cancel := context.WithTimeout(context.Background(), devTimeout)
				outDev, devUsage, err = events.RunWithContext(ctx, bus, parsedFeatureName, events.RoleDev, &cfg.Dev, finalPrompt)
				if ctx.Err() == context.DeadlineExceeded {
					err = fmt.Errorf("dev agent exceeded %s — the model may not fit in memory; try a smaller model or a lower context_window", devTimeout)
				}
				cancel()
			} else {
				outDev, devUsage, err = events.Run(bus, parsedFeatureName, events.RoleDev, &cfg.Dev, finalPrompt)
			}
			tracker.AddTokens(devUsage.PromptTokens, devUsage.EvalTokens)
			if err != nil {
				fmt.Printf("⚠️ Dev Agent run error: %v\n", err)
			}

			// Extract and write generated code files if agent is llama_cpp
			if cfg.Dev.Agent == "llama_cpp" {
				parseAndWriteGeneratedFiles(outDev, targetSubfolder, parsedFeatureName, bus)
			}

			// ── PHASE 2: QA VERIFICATION (PARALLEL AUDIT + TESTS) ──
			UpdateState(StageQA, parsedFeatureName, retry, tracker, bus)
			fmt.Println("🛡️ Running Security Audit & Test Suite in parallel...")

			// Launch security audit and go test concurrently via goroutines
			auditCh := make(chan error, 1)
			testCh := make(chan *qa.TestResult, 1)

			go func() { 
				if cfg.ForceRegression {
					auditCh <- qa.AuditGeneratedCode("workspace", cfg.QAIgnore, cfg.QARules)
				} else {
					auditCh <- qa.AuditGeneratedCode(targetSubfolder, nil, cfg.QARules)
				}
			}()
			
			go func() { 
				if cfg.ForceRegression {
					testCh <- qa.RunTests("workspace", true, cfg.QAIgnore) 
				} else {
					testCh <- qa.RunTests(targetSubfolder, false, nil)
				}
			}()

			auditErr := <-auditCh
			testResult := <-testCh

			// Evaluate combined results from both goroutines
			var qaErrors []string
			result := events.QAResult{
				Target:     targetSubfolder,
				Attempt:    retry + 1,
				MaxRetries: MaxRetries,
			}
			if auditErr != nil {
				qaErrors = append(qaErrors, fmt.Sprintf("SECURITY AUDIT FAILURE: %v", auditErr))
				result.AuditError = events.TruncateText(auditErr.Error(), events.MaxErrorText)
			}
			if testResult.Err != nil {
				qaErrors = append(qaErrors, fmt.Sprintf("TEST FAILURE:\n%s", string(testResult.Output)))
				// Capped: a verbose regression dump is megabytes, and qa_error.log
				// on disk keeps the full text for the Dev agent to self-heal from.
				result.TestError = events.TruncateText(string(testResult.Output), events.MaxErrorText)
			}
			result.Passed = len(qaErrors) == 0
			bus.Emit(events.KindQAResult, parsedFeatureName, result)

			if len(qaErrors) > 0 {
				fmt.Printf("⚠️ QA failed on attempt %d! Writing combined errors to qa_error.log for AI self-healing...\n", retry+1)
				_ = os.WriteFile("workspace/qa_error.log", []byte(strings.Join(qaErrors, "\n\n---\n\n")), 0644)
				tracker.IncrRetries()
				bus.Emit(events.KindRetry, parsedFeatureName, events.Retry{
					Attempt:    retry + 1,
					MaxRetries: MaxRetries,
				})
				time.Sleep(2 * time.Second)
			} else {
				fmt.Println("🎉 Excellent! Security audit passed and 100% of QA Test Suite passed.")
				_ = os.Remove("workspace/qa_error.log")
				if retry > 0 {
					tracker.MarkHealing()
				}
				success = true
				break
			}
		}

		if success {
			break
		}

		if delegation < maxDelegations {
			qaLogs, _ := os.ReadFile("workspace/qa_error.log")
			dodContent, _ := os.ReadFile("memory/definitions_of_done.md")
			_, featureName := parseDoDTarget(string(dodContent))

			// Activate Delegation Protocol
			UpdateState(StageBARefactor, featureName, delegation, tracker, bus)
			bus.Emit(events.KindDelegation, featureName, events.Delegation{
				Cycle:    delegation,
				MaxCycle: maxDelegations,
			})
			fmt.Println("🔄 Activating Delegation Protocol: BA Agent rewriting requirements...")

			// Context Reset: clear previous failure context for the new loop iteration
			fmt.Println("🧹 Clearing agent context for a clean loop...")
			_ = os.RemoveAll(".antigravitycli")
			_ = os.RemoveAll(".claude")

			baPrompt := fmt.Sprintf(`You are being delegated a failing task. Analyze why the engineer failed to build the code based on the compilation logs. Rewrite the requirements inside memory/definitions_of_done.md to clarify ambiguity, fix structural holes, or split the task safely.

=== COMPILATION LOGS ===
%s

=== CURRENT REQUIREMENTS ===
%s

Output ONLY the strict markdown checklist content. Do not include any chat filler or explanations.`, string(qaLogs), string(dodContent))

			outBA, baUsage, errBA := events.Run(bus, featureName, events.RoleBA, &cfg.BA, baPrompt)
			tracker.AddTokens(baUsage.PromptTokens, baUsage.EvalTokens)
			if errBA != nil {
				fmt.Printf("⚠️ BA Agent failed during delegation: %v\n", errBA)
			} else {
				_ = os.WriteFile("memory/definitions_of_done.md", []byte(outBA), 0644)
			}
		}
	}

	dodContent, _ := os.ReadFile("memory/definitions_of_done.md")
	targetSubfolder, parsedFeatureName := parseDoDTarget(string(dodContent))

	if !success {
		// Emitted before the fatal exit so a monitor sees why the stream ends.
		events.EmitRunFinished(bus, parsedFeatureName, tracker, false,
			fmt.Sprintf("QA failed after %d self-healing loops and %d delegation cycles", MaxRetries, maxDelegations))
		log.Fatalf("❌ [Harness Aborted] Agent attempted %d self-healing loops and %d delegation cycles but failed QA. Manual intervention required!", MaxRetries, maxDelegations)
	}

	// =======================================================
	// PHASE 3: HUMAN-IN-THE-LOOP (HITL INTERCEPTOR)
	// =======================================================
	UpdateState(StageHITL, parsedFeatureName, 0, tracker, bus)
	bus.Emit(events.KindHITLPrompt, parsedFeatureName, events.HITLPrompt{
		Target:         targetSubfolder,
		TimeoutSeconds: int(hitlTimeout / time.Second),
	})
	fmt.Printf("🚧 [HITL GATE] Code passed QA. Do you APPROVE integrating this into the System Blueprint? (y/n) [Auto-yes in %s]: ", hitlTimeout)

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
	bus.Emit(events.KindHITLResolved, parsedFeatureName, events.HITLResolved{
		Approved: approved,
		Auto:     auto,
	})
	if !approved {
		fmt.Println("🛑 Disapproved! Terminating pipeline without updating memory.")
		events.EmitRunFinished(bus, parsedFeatureName, tracker, false, "rejected at HITL gate")
		os.Exit(1)
	}

	// =======================================================
	// PHASE 4: DEVOPS & LOCAL OUTCOME COMPILATION
	// =======================================================
	UpdateState(StageDevOps, parsedFeatureName, 0, tracker, bus)
	fmt.Printf("📝 Invoking local %s agent (%s) to construct deployment documentation...\n", cfg.DevOps.Agent, cfg.DevOps.ModelName)

	generateReleaseNotes(&cfg.DevOps, targetSubfolder, parsedFeatureName, tracker, bus)

	// =======================================================
	// PHASE 5: AUTO-DOCUMENTATION (OPENWIKI)
	// =======================================================
	fmt.Println("\n==============================================")
	fmt.Println("PHASE 5: AUTO-DOCUMENTATION (OPENWIKI)")
	fmt.Println("==============================================")
	if err := docs.GenerateDocumentation(cfg.DevOps.ModelName); err != nil {
		fmt.Printf("⚠️ OpenWiki integration encountered an error: %v\n", err)
	}

	// =======================================================
	// FINALIZE: Memory Progression (fully sequential)
	// =======================================================
	UpdateState(StageCompact, parsedFeatureName, 0, tracker, bus)
	memory.UpdateSystemMemory(&cfg.DevOps, tracker, bus, parsedFeatureName)
	memory.CompactSystemMemory(&cfg.DevOps, tracker, bus, parsedFeatureName)

	UpdateState(StageDone, parsedFeatureName, 0, tracker, bus)
	fmt.Println("\n🎯 SPRINT PIPELINE RUN COMPLETE. Check your /workspace folder for final artifacts!")
}

// containPath validates that an LLM-emitted file path resolves to a location
// strictly inside targetSubfolder. It strips a leading slash (the dev prompt
// sometimes uses "/workspace/...") and rejects any path that escapes the folder
// via "..". This is a security boundary: the QA audit only scans targetSubfolder,
// so files written elsewhere would silently bypass the scan and could clobber
// files anywhere on disk.
func containPath(targetSubfolder, raw string) (string, bool) {
	cleaned := filepath.Clean(strings.TrimPrefix(raw, "/"))
	base := filepath.Clean(targetSubfolder)
	rel, err := filepath.Rel(base, cleaned)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", false
	}
	return cleaned, true
}

// parseAndWriteGeneratedFiles extracts file paths and code blocks or SEARCH/REPLACE blocks from the raw LLM output and writes them to the workspace.
// Writes are contained to targetSubfolder; paths that escape it are rejected.
func parseAndWriteGeneratedFiles(output, targetSubfolder, task string, bus *events.Bus) {
	lines := strings.Split(output, "\n")
	var currentFile string
	var currentContent []string

	inCodeBlock := false
	inSearchBlock := false
	inReplaceBlock := false

	var searchContent []string
	var replaceContent []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "### FILE: ") {
			raw := strings.TrimSpace(strings.TrimPrefix(trimmed, "### FILE: "))
			currentContent = []string{}
			searchContent = []string{}
			replaceContent = []string{}
			inCodeBlock = false
			inSearchBlock = false
			inReplaceBlock = false
			if safe, ok := containPath(targetSubfolder, raw); ok {
				currentFile = safe
			} else {
				currentFile = "" // reject: skips the block until the next ### FILE:
				bus.Emit(events.KindFileRejected, task, events.FileRejected{
					Path:   raw,
					Target: targetSubfolder,
					Reason: "path escapes the target folder",
				})
				fmt.Printf("🛑 Rejected generated file outside target folder %q: %s\n", targetSubfolder, raw)
			}
			continue
		}

		if currentFile == "" {
			continue
		}

		// Backward compatibility: Full file overwrite
		if strings.HasPrefix(trimmed, "```") && !inSearchBlock && !inReplaceBlock {
			if !inCodeBlock {
				inCodeBlock = true
				continue
			} else {
				// End of code block, write file
				inCodeBlock = false
				dir := filepath.Dir(currentFile)
				_ = os.MkdirAll(dir, 0755)
				_ = os.WriteFile(currentFile, []byte(strings.Join(currentContent, "\n")), 0644)
				bus.Emit(events.KindFileWritten, task, events.FileWritten{
					Path:   currentFile,
					Action: "write",
				})
				fmt.Printf("📝 Wrote generated file: %s\n", currentFile)
				currentFile = "" // reset
				continue
			}
		}

		// Search/Replace Block Logic
		if trimmed == "<<<<" {
			inSearchBlock = true
			searchContent = []string{}
			continue
		}
		if trimmed == "====" && inSearchBlock {
			inSearchBlock = false
			inReplaceBlock = true
			replaceContent = []string{}
			continue
		}
		if trimmed == ">>>>" && inReplaceBlock {
			inReplaceBlock = false
			
			// Apply the replacement
			content, err := os.ReadFile(currentFile)
			if err == nil {
				oldCode := strings.Join(searchContent, "\n")
				newCode := strings.Join(replaceContent, "\n")
				
				// Optional: trim trailing newlines for safety
				fileContent := string(content)
				if strings.Contains(fileContent, oldCode) {
					updatedContent := strings.Replace(fileContent, oldCode, newCode, 1)
					_ = os.WriteFile(currentFile, []byte(updatedContent), 0644)
					bus.Emit(events.KindFileWritten, task, events.FileWritten{
						Path:   currentFile,
						Action: "patch",
					})
					fmt.Printf("📝 Applied patch to file: %s\n", currentFile)
				} else {
					fmt.Printf("⚠️ Failed to apply patch to %s: search block not found.\n", currentFile)
				}
			} else {
				fmt.Printf("⚠️ Failed to read file for patching: %s\n", currentFile)
			}
			continue
		}

		// Buffer content based on state
		if inSearchBlock {
			searchContent = append(searchContent, line)
		} else if inReplaceBlock {
			replaceContent = append(replaceContent, line)
		} else if inCodeBlock {
			currentContent = append(currentContent, line)
		}
	}
}
