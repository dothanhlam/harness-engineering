package pipeline

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dothanhlam/harness-app/internal/config"
	"github.com/dothanhlam/harness-app/internal/memory"
	"github.com/dothanhlam/harness-app/internal/qa"
	"github.com/dothanhlam/harness-app/internal/telemetry"
)

// RunCoreHarnessLoop runs the core Development → QA → HITL → DevOps → Memory loop.
// Uses goroutines for parallel QA (audit ∥ tests) and async Linear ticket updates.
// Memory progression is fully sequential per design decision.
func RunCoreHarnessLoop(cfg config.Config, tracker *telemetry.Tracker) {
	mem0Client := memory.NewMem0Client(cfg.Mem0.BaseURL, cfg.Mem0.AgentID)
	// =======================================================
	// CORE LOOP: DEV 🔀 QA (SELF-HEALING) & DELEGATION
	// =======================================================
	_ = os.Remove("workspace/qa_error.log")
	success := false
	maxDelegations := 1

	for delegation := 0; delegation <= maxDelegations; delegation++ {
		for retry := 0; retry < MaxRetries; retry++ {
			// ── PHASE 1: DEVELOPMENT / REPAIR ──
			UpdateState(StageDev, retry, tracker)
			fmt.Printf("🤖 Activating %s CLI for Code Generation/Repair...\n", cfg.Dev.Agent)

			devPrompt, err := os.ReadFile(".agents/antigravity_dev_prompt.md")
			if err != nil {
				log.Fatalf("❌ Missing configuration file: .agents/antigravity_dev_prompt.md")
			}

			dodContent, _ := os.ReadFile("memory/definitions_of_done.md")
			memories, searchErr := mem0Client.SearchMemories(string(dodContent), 3)
			contextStr := ""
			if searchErr == nil && len(memories) > 0 {
				contextStr = "\n\n=== SYSTEM ARCHITECTURE CONTEXT FROM MEM0 ===\n- " + strings.Join(memories, "\n- ")
			}

			_, err = cfg.Dev.Execute(string(devPrompt) + contextStr)
			if err != nil {
				fmt.Printf("⚠️ Dev Agent run error: %v\n", err)
			}

			// ── PHASE 2: QA VERIFICATION (PARALLEL AUDIT + TESTS) ──
			UpdateState(StageQA, retry, tracker)
			fmt.Println("🛡️ Running Security Audit & Test Suite in parallel...")

			// Launch security audit and go test concurrently via goroutines
			auditCh := make(chan error, 1)
			testCh := make(chan *qa.TestResult, 1)

			go func() { auditCh <- qa.AuditGeneratedCode("workspace") }()
			go func() { testCh <- qa.RunTests("./workspace/...") }()

			auditErr := <-auditCh
			testResult := <-testCh

			// Evaluate combined results from both goroutines
			var qaErrors []string
			if auditErr != nil {
				qaErrors = append(qaErrors, fmt.Sprintf("SECURITY AUDIT FAILURE: %v", auditErr))
			}
			if testResult.Err != nil {
				qaErrors = append(qaErrors, fmt.Sprintf("TEST FAILURE:\n%s", string(testResult.Output)))
			}

			if len(qaErrors) > 0 {
				fmt.Printf("⚠️ QA failed on attempt %d! Writing combined errors to qa_error.log for AI self-healing...\n", retry+1)
				_ = os.WriteFile("workspace/qa_error.log", []byte(strings.Join(qaErrors, "\n\n---\n\n")), 0644)
				tracker.IncrRetries()
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
			// Activate Delegation Protocol
			UpdateState(StageBARefactor, delegation, tracker)
			fmt.Println("🔄 Activating Delegation Protocol: BA Agent rewriting requirements...")

			qaLogs, _ := os.ReadFile("workspace/qa_error.log")
			dodContent, _ := os.ReadFile("memory/definitions_of_done.md")

			baPrompt := fmt.Sprintf(`You are being delegated a failing task. Analyze why the engineer failed to build the code based on the compilation logs. Rewrite the requirements inside memory/definitions_of_done.md to clarify ambiguity, fix structural holes, or split the task safely.

=== COMPILATION LOGS ===
%s

=== CURRENT REQUIREMENTS ===
%s

Output ONLY the strict markdown checklist content. Do not include any chat filler or explanations.`, string(qaLogs), string(dodContent))

			outBA, errBA := cfg.BA.Execute(baPrompt)
			if errBA != nil {
				fmt.Printf("⚠️ BA Agent failed during delegation: %v\n", errBA)
			} else {
				_ = os.WriteFile("memory/definitions_of_done.md", []byte(outBA), 0644)
			}
		}
	}

	if !success {
		log.Fatalf("❌ [Harness Aborted] Agent attempted %d self-healing loops and %d delegation cycles but failed QA. Manual intervention required!", MaxRetries, maxDelegations)
	}

	// =======================================================
	// PHASE 3: HUMAN-IN-THE-LOOP (HITL INTERCEPTOR)
	// =======================================================
	UpdateState(StageHITL, 0, tracker)
	fmt.Print("🚧 [HITL GATE] Code passed QA. Do you APPROVE integrating this into the System Blueprint? (y/n) [Auto-yes in 30s]: ")

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
		fmt.Println("🛑 Disapproved! Terminating pipeline without updating memory.")
		os.Exit(1)
	}

	// =======================================================
	// PHASE 4: DEVOPS & LOCAL OUTCOME COMPILATION
	// =======================================================
	UpdateState(StageDevOps, 0, tracker)
	fmt.Printf("📝 Invoking local %s agent (%s) to construct deployment documentation...\n", cfg.DevOps.Agent, cfg.DevOps.ModelName)

	// Channel for non-blocking Linear ticket update goroutine
	linearDone := make(chan struct{})
	linearStarted := false

	dodContent, errReadDoD := os.ReadFile("memory/definitions_of_done.md")
	var targetSubfolder, ticketID, parsedFeatureName string
	if errReadDoD == nil {
		for _, line := range strings.Split(string(dodContent), "\n") {
			if strings.HasPrefix(line, "- Target Subfolder: ") {
				targetSubfolder = strings.TrimSpace(strings.TrimPrefix(line, "- Target Subfolder: "))
				parsedFeatureName = filepath.Base(targetSubfolder)
			} else if strings.HasPrefix(line, "- Ticket ID: ") {
				ticketID = strings.TrimSpace(strings.TrimPrefix(line, "- Ticket ID: "))
			} else if strings.HasPrefix(line, "# TASK: ") && parsedFeatureName == "" {
				parsedFeatureName = strings.TrimSpace(strings.TrimPrefix(line, "# TASK: "))
			}
		}
	}

	if targetSubfolder != "" {
		featureFiles, _ := os.ReadDir(targetSubfolder)
		var allCode string
		for _, ff := range featureFiles {
			if !ff.IsDir() && strings.HasSuffix(ff.Name(), ".go") {
				content, _ := os.ReadFile(fmt.Sprintf("%s/%s", targetSubfolder, ff.Name()))
				allCode += string(content) + "\n"
			}
		}

		if allCode != "" {
			sysPrompt := fmt.Sprintf("You are a deployment release manager. Generate a short, bulleted markdown release note based on the provided Go code for the feature '%s'. Keep it brief. Be extremely concise. Return bullet points only. Limit your response to under 150 words. Do not write filler structural prose.", parsedFeatureName)
			fullPrompt := fmt.Sprintf("SYSTEM INSTRUCTIONS:\n%s\n\nUSER INPUT:\n%s", sysPrompt, allCode)
			
			var releaseNotes string
			var errDevOps error
			if cfg.DevOps.Agent == "ollama" {
				ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
				defer cancel()
				releaseNotes, errDevOps = cfg.DevOps.ExecuteWithContext(ctx, fullPrompt)
				if errDevOps != nil {
					fmt.Println("⚠️ [OLLAMA THERMAL THROTTLING] DevOps agent timed out. Gracefully falling back to save CPU cycles...")
					releaseNotes = "- DevOps auto-generation aborted (thermal fallback).\n- Check commits for details."
					errDevOps = nil
				}
			} else {
				releaseNotes, errDevOps = cfg.DevOps.Execute(fullPrompt)
			}
			
			notePath := fmt.Sprintf("%s/RELEASE_NOTES.md", targetSubfolder)
			if errDevOps != nil {
				fmt.Printf("⚠️ DevOps Agent communication failed for %s: %v\n", parsedFeatureName, errDevOps)
			} else {
				_ = os.WriteFile(notePath, []byte(releaseNotes), 0644)
				fmt.Printf("📝 Generated %s automatically using local resources.\n", notePath)

				if ticketID != "" && cfg.DevOps.MCPConfig != "" {
					// Fire Linear ticket update as a non-blocking goroutine
					linearStarted = true
					go func() {
						defer close(linearDone)
						fmt.Printf("🚀 Triggering dev agent to update Linear ticket %s...\n", ticketID)
						linearPrompt := fmt.Sprintf("Tests passed and feature '%s' is complete. Please update the Linear ticket %s with the following release notes:\n%s", parsedFeatureName, ticketID, releaseNotes)
						outLinear, errLinear := cfg.Dev.Execute(linearPrompt)
						if errLinear != nil {
							fmt.Printf("⚠️ Warning: Failed to update Linear ticket %s: %v\n", ticketID, errLinear)
						} else {
							fmt.Printf("✅ Successfully updated Linear ticket %s.\n%s\n", ticketID, outLinear)
						}
					}()
				} else if ticketID != "" {
					fmt.Println("ℹ️ Linear MCP config missing. Skipping Linear ticket update.")
				} else {
					fmt.Println("ℹ️ No Ticket ID configured for this task. Skipping Linear ticket update.")
				}
			}
		}
	} else {
		// Fallback for legacy single-task mode: iterate over entire workspace
		entries, errRead := os.ReadDir("workspace")
		if errRead == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				feature := entry.Name()

				featureFiles, _ := os.ReadDir(fmt.Sprintf("workspace/%s", feature))
				var allCode string
				for _, ff := range featureFiles {
					if !ff.IsDir() && strings.HasSuffix(ff.Name(), ".go") {
						content, _ := os.ReadFile(fmt.Sprintf("workspace/%s/%s", feature, ff.Name()))
						allCode += string(content) + "\n"
					}
				}

				if allCode == "" {
					continue
				}

				sysPrompt := fmt.Sprintf("You are a deployment release manager. Generate a short, bulleted markdown release note based on the provided Go code for the feature '%s'. Keep it brief. Be extremely concise. Return bullet points only. Limit your response to under 150 words. Do not write filler structural prose.", feature)
				fullPrompt := fmt.Sprintf("SYSTEM INSTRUCTIONS:\n%s\n\nUSER INPUT:\n%s", sysPrompt, allCode)
				
				var releaseNotes string
				var errDevOps error
				if cfg.DevOps.Agent == "ollama" {
					ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
					defer cancel()
					releaseNotes, errDevOps = cfg.DevOps.ExecuteWithContext(ctx, fullPrompt)
					if errDevOps != nil {
						fmt.Println("⚠️ [OLLAMA THERMAL THROTTLING] DevOps agent timed out. Gracefully falling back to save CPU cycles...")
						releaseNotes = "- DevOps auto-generation aborted (thermal fallback).\n- Check commits for details."
						errDevOps = nil
					}
				} else {
					releaseNotes, errDevOps = cfg.DevOps.Execute(fullPrompt)
				}

				notePath := fmt.Sprintf("workspace/%s/RELEASE_NOTES.md", feature)
				if errDevOps != nil {
					fmt.Printf("⚠️ DevOps Agent communication failed for %s: %v\n", feature, errDevOps)
				} else {
					_ = os.WriteFile(notePath, []byte(releaseNotes), 0644)
					fmt.Printf("📝 Generated %s automatically using local resources.\n", notePath)
					fmt.Println("ℹ️ Running in legacy workspace mode without explicit ticket ID. Skipping Linear ticket update.")
				}
			}
		}
	}

	// =======================================================
	// FINALIZE: Memory Progression (fully sequential)
	// =======================================================
	UpdateState(StageCompact, 0, tracker)
	memory.UpdateSystemMemory(mem0Client, &cfg.DevOps)

	// Wait for async Linear update if it was started
	if linearStarted {
		<-linearDone
	}

	UpdateState(StageDone, 0, tracker)
	fmt.Println("\n🎯 SPRINT PIPELINE RUN COMPLETE. Check your /workspace folder for final artifacts!")
}
