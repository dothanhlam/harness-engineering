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
	"github.com/dothanhlam/harness-engineering/internal/memory"
	"github.com/dothanhlam/harness-engineering/internal/qa"
	"github.com/dothanhlam/harness-engineering/internal/telemetry"
)

// RunCoreHarnessLoop runs the core Development → QA → HITL → DevOps → Memory loop.
// Uses goroutines for parallel QA (audit ∥ tests).
// Memory progression is fully sequential per design decision.
func RunCoreHarnessLoop(cfg config.Config, tracker *telemetry.Tracker) {
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
			UpdateState(StageDev, retry, tracker)
			fmt.Printf("🤖 Activating %s CLI for Code Generation/Repair...\n", cfg.Dev.Agent)

			devPrompt, err := os.ReadFile(".agents/antigravity_dev_prompt.md")
			if err != nil {
				log.Fatalf("❌ Missing configuration file: .agents/antigravity_dev_prompt.md")
			}

			// Context Injection for local LLMs (they lack filesystem read tools)
			var finalPrompt string
			
			// Parse the feature name from definitions_of_done.md to replace placeholders
			var parsedFeatureName string
			var targetSubfolder string
			dod, errDod := os.ReadFile("memory/definitions_of_done.md")
			if errDod == nil {
				for _, line := range strings.Split(string(dod), "\n") {
					if strings.HasPrefix(line, "- Target Subfolder: ") {
						targetSubfolder = strings.TrimSpace(strings.TrimPrefix(line, "- Target Subfolder: "))
						parsedFeatureName = filepath.Base(targetSubfolder)
					} else if strings.HasPrefix(line, "# TASK: ") && parsedFeatureName == "" {
						parsedFeatureName = strings.TrimSpace(strings.TrimPrefix(line, "# TASK: "))
					}
				}
			}
			
			if parsedFeatureName == "" {
				parsedFeatureName = "default_task"
			}
			if targetSubfolder == "" {
				targetSubfolder = fmt.Sprintf("workspace/%s", parsedFeatureName)
			}

			if cfg.Dev.Agent == "llama_cpp" {
				var sb strings.Builder
				
				// Replace "feature_name" placeholder in the prompt template
				promptStr := string(devPrompt)
				if parsedFeatureName != "" {
					promptStr = strings.ReplaceAll(promptStr, "feature_name", parsedFeatureName)
					promptStr = strings.ReplaceAll(promptStr, "filename", parsedFeatureName)
				}
				sb.WriteString(promptStr)
				
				if errDod == nil {
					sb.WriteString("\n\n=== CONTEXT: memory/definitions_of_done.md ===\n")
					sb.WriteString(string(dod))
				}
				
				blueprint, err := os.ReadFile("memory/system_blueprint.md")
				if err == nil {
					sb.WriteString("\n\n=== CONTEXT: memory/system_blueprint.md ===\n")
					sb.WriteString(string(blueprint))
				}
				
				qaErr, err := os.ReadFile("workspace/qa_error.log")
				if err == nil {
					sb.WriteString("\n\n=== CONTEXT: workspace/qa_error.log (Fix these errors!) ===\n")
					sb.WriteString(string(qaErr))
				}
				
				finalPrompt = sb.String()
			} else {
				finalPrompt = string(devPrompt)
			}

			outDev, devUsage, err := cfg.Dev.Execute(finalPrompt)
			tracker.AddTokens(devUsage.PromptTokens, devUsage.EvalTokens)
			if err != nil {
				fmt.Printf("⚠️ Dev Agent run error: %v\n", err)
			}

			// Extract and write generated code files if agent is llama_cpp
			if cfg.Dev.Agent == "llama_cpp" {
				parseAndWriteGeneratedFiles(outDev)
			}

			// ── PHASE 2: QA VERIFICATION (PARALLEL AUDIT + TESTS) ──
			UpdateState(StageQA, retry, tracker)
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

			// Context Reset: clear previous failure context for the new loop iteration
			fmt.Println("🧹 Clearing agent context for a clean loop...")
			_ = os.RemoveAll(".antigravitycli")
			_ = os.RemoveAll(".claude")

			qaLogs, _ := os.ReadFile("workspace/qa_error.log")
			dodContent, _ := os.ReadFile("memory/definitions_of_done.md")

			baPrompt := fmt.Sprintf(`You are being delegated a failing task. Analyze why the engineer failed to build the code based on the compilation logs. Rewrite the requirements inside memory/definitions_of_done.md to clarify ambiguity, fix structural holes, or split the task safely.

=== COMPILATION LOGS ===
%s

=== CURRENT REQUIREMENTS ===
%s

Output ONLY the strict markdown checklist content. Do not include any chat filler or explanations.`, string(qaLogs), string(dodContent))

			outBA, baUsage, errBA := cfg.BA.Execute(baPrompt)
			tracker.AddTokens(baUsage.PromptTokens, baUsage.EvalTokens)
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

	dodContent, errReadDoD := os.ReadFile("memory/definitions_of_done.md")
	var targetSubfolder, parsedFeatureName string
	if errReadDoD == nil {
		for _, line := range strings.Split(string(dodContent), "\n") {
			if strings.HasPrefix(line, "- Target Subfolder: ") {
				targetSubfolder = strings.TrimSpace(strings.TrimPrefix(line, "- Target Subfolder: "))
				parsedFeatureName = filepath.Base(targetSubfolder)
			} else if strings.HasPrefix(line, "# TASK: ") && parsedFeatureName == "" {
				parsedFeatureName = strings.TrimSpace(strings.TrimPrefix(line, "# TASK: "))
			}
		}
	}
	
	if parsedFeatureName == "" {
		parsedFeatureName = "default_task"
	}
	if targetSubfolder == "" {
		targetSubfolder = fmt.Sprintf("workspace/%s", parsedFeatureName)
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
			if cfg.DevOps.Agent == "llama_cpp" {
				ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
				defer cancel()
				var doUsage agent.TokenUsage
				releaseNotes, doUsage, errDevOps = cfg.DevOps.ExecuteWithContext(ctx, fullPrompt)
				tracker.AddTokens(doUsage.PromptTokens, doUsage.EvalTokens)
				if errDevOps != nil {
					fmt.Printf("⚠️ [%s THERMAL THROTTLING] DevOps agent timed out. Gracefully falling back to save CPU cycles...\n", strings.ToUpper(cfg.DevOps.Agent))
					releaseNotes = "- DevOps auto-generation aborted (thermal fallback).\n- Check commits for details."
					errDevOps = nil
				}
			} else {
				var doUsage agent.TokenUsage
				releaseNotes, doUsage, errDevOps = cfg.DevOps.Execute(fullPrompt)
				tracker.AddTokens(doUsage.PromptTokens, doUsage.EvalTokens)
			}

			notePath := fmt.Sprintf("%s/RELEASE_NOTES.md", targetSubfolder)
			if errDevOps != nil {
				fmt.Printf("⚠️ DevOps Agent communication failed for %s: %v\n", parsedFeatureName, errDevOps)
			} else {
				_ = os.WriteFile(notePath, []byte(releaseNotes), 0644)
				fmt.Printf("📝 Generated %s automatically using local resources.\n", notePath)

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
				if cfg.DevOps.Agent == "llama_cpp" {
					ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
					defer cancel()
					var doUsage agent.TokenUsage
					releaseNotes, doUsage, errDevOps = cfg.DevOps.ExecuteWithContext(ctx, fullPrompt)
					tracker.AddTokens(doUsage.PromptTokens, doUsage.EvalTokens)
					if errDevOps != nil {
						fmt.Printf("⚠️ [%s THERMAL THROTTLING] DevOps agent timed out. Gracefully falling back to save CPU cycles...\n", strings.ToUpper(cfg.DevOps.Agent))
						releaseNotes = "- DevOps auto-generation aborted (thermal fallback).\n- Check commits for details."
						errDevOps = nil
					}
				} else {
					var doUsage agent.TokenUsage
					releaseNotes, doUsage, errDevOps = cfg.DevOps.Execute(fullPrompt)
					tracker.AddTokens(doUsage.PromptTokens, doUsage.EvalTokens)
				}

				notePath := fmt.Sprintf("workspace/%s/RELEASE_NOTES.md", feature)
				if errDevOps != nil {
					fmt.Printf("⚠️ DevOps Agent communication failed for %s: %v\n", feature, errDevOps)
				} else {
					_ = os.WriteFile(notePath, []byte(releaseNotes), 0644)
					fmt.Printf("📝 Generated %s automatically using local resources.\n", notePath)
				}
			}
		}
	}

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
	UpdateState(StageCompact, 0, tracker)
	memory.UpdateSystemMemory(&cfg.DevOps, tracker)
	memory.CompactSystemMemory(&cfg.DevOps, tracker)

	UpdateState(StageDone, 0, tracker)
	fmt.Println("\n🎯 SPRINT PIPELINE RUN COMPLETE. Check your /workspace folder for final artifacts!")
}

// parseAndWriteGeneratedFiles extracts file paths and code blocks or SEARCH/REPLACE blocks from the raw LLM output and writes them to the workspace.
func parseAndWriteGeneratedFiles(output string) {
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
			currentFile = strings.TrimSpace(strings.TrimPrefix(trimmed, "### FILE: "))
			currentContent = []string{}
			searchContent = []string{}
			replaceContent = []string{}
			inCodeBlock = false
			inSearchBlock = false
			inReplaceBlock = false
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
