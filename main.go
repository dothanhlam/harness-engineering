package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ─────────────────────────────────────────────
// PIPELINE STAGE CONSTANTS
// ─────────────────────────────────────────────

type Stage string

const (
	StageDev     Stage = "DEV_CODING"
	StageQA      Stage = "QA_TESTING"
	StageHITL    Stage = "HUMAN_IN_THE_LOOP"
	StageCompact Stage = "MEMORY_COMPACTION"
	StageDevOps  Stage = "DEVOPS_DELIVER"
	StageDone    Stage = "COMPLETED"
)

const MaxRetries = 3

// WorkflowState is the persisted pipeline state written to workspace/state.json.
type WorkflowState struct {
	TaskID       string    `json:"task_id"`
	CurrentStage Stage     `json:"current_stage"`
	RetryCount   int       `json:"retry_count"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Telemetry struct {
	TotalDurationSeconds float64  `json:"total_duration_seconds"`
	StagesExecuted       []string `json:"stages_executed"`
	TotalRetriesUsed     int      `json:"total_retries_used"`
	CodeHealingSuccess   bool     `json:"code_healing_success"`
	LinesOfCodeGenerated int      `json:"lines_of_code_generated"`
	Timestamp            string   `json:"timestamp"`
}

var (
	pipelineTelemetry Telemetry
	pipelineStartTime time.Time
)

// EpicPipeline holds the decomposed tasks for an epic.
type EpicPipeline struct {
	SubTasks []struct {
		Name         string `json:"task_name"`
		TargetFolder string `json:"target_subfolder"`
		PromptSpecs  string `json:"prompt_specifications"`
		TicketID     string `json:"ticket_id,omitempty"`
	} `json:"sub_tasks"`
}

// updateState writes the current pipeline stage to workspace/state.json.
func updateState(stage Stage, retry int) {
	pipelineTelemetry.StagesExecuted = append(pipelineTelemetry.StagesExecuted, fmt.Sprintf("%s (%s)", stage, time.Now().Format(time.RFC3339)))
	state := WorkflowState{
		TaskID:       "cti_modular_self_healing",
		CurrentStage: stage,
		RetryCount:   retry,
		UpdatedAt:    time.Now(),
	}
	file, _ := json.MarshalIndent(state, "", "  ")
	_ = os.WriteFile("workspace/state.json", file, 0644)
	fmt.Printf("\n🔄 [HARNESS STATE] -> %s (Attempt: %d/%d)\n", stage, retry+1, MaxRetries)
}

// compactSystemMemory uses AI to optimize the system_blueprint.md if it gets too large
func compactSystemMemory(agent, modelName string) {
	blueprintPath := "memory/system_blueprint.md"
	data, err := os.ReadFile(blueprintPath)
	if err != nil || len(data) < 3000 {
		return
	}

	fmt.Println("🧹 [MEMORY COMPACTION] System memory is getting too long. Activating AI optimization...")
	sysPrompt := `You are an Enterprise System Architect. The system blueprint memory is getting too long. 
Review the entire text and compress older historical feature logs into a single unified 'Current System Architecture Map' section. 
Keep the latest 2 features fully intact, but summarize all previous ones into architectural line items. Return ONLY the new compact markdown.`

	compacted, err := callOllama(agent, modelName, sysPrompt, string(data))
	if err == nil {
		_ = os.WriteFile(blueprintPath, []byte(compacted), 0644)
		fmt.Println("✅ [MEMORY COMPACTION] Successfully optimized Memory context window!")
	}
}

// updateSystemMemory progressively analyzes architectural correlations and archives features.
func updateSystemMemory(agent, modelName string) {
	fmt.Println("🧠 [PROGRESSIVE MEMORY] Analyzing modular architectural correlations...")
	oldBlueprint, _ := os.ReadFile("memory/system_blueprint.md")
	currentDoD, _ := os.ReadFile("memory/definitions_of_done.md")

	sysPrompt := `You are an Enterprise System Architect. Analyze the new requirement against the existing blueprint.
Identify structural dependencies, package reusability, or architectural correlations. Return ONLY the concise markdown log.`

	userPrompt := fmt.Sprintf("=== REQS ===\n%s\n=== BLUEPRINT ===\n%s", string(currentDoD), string(oldBlueprint))
	correlations, err := callOllama(agent, modelName, sysPrompt, userPrompt)
	if err == nil {
		newContent := fmt.Sprintf("%s\n\n## [ARCHIVED FEATURE - %s]\n%s", string(oldBlueprint), time.Now().Format("2006-01-02 15:04"), correlations)
		_ = os.WriteFile("memory/system_blueprint.md", []byte(newContent), 0644)
		fmt.Println("✅ [PROGRESSIVE MEMORY] System architecture map synchronized.")
	}
}

// callOllama sends a chat request to a configurable Ollama API endpoint.
// callOllama sends a prompt to the local CLI binary (e.g., ollama).
func callOllama(agent, modelName, systemPrompt, userPrompt string) (string, error) {
	fullPrompt := fmt.Sprintf("SYSTEM INSTRUCTIONS:\n%s\n\nUSER INPUT:\n%s", systemPrompt, userPrompt)

	cmd := exec.Command(agent, "run", modelName, fullPrompt)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ollama CLI error: %v, stderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(out.String()), nil
}

func countGeneratedLines() int {
	var count int
	_ = filepath.Walk("workspace", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			content, err := os.ReadFile(path)
			if err == nil {
				count += len(strings.Split(string(content), "\n"))
			}
		}
		return nil
	})
	return count
}

// runCoreHarnessLoop runs the core Development, QA, HITL, DevOps, and Memory Progression loop
func runCoreHarnessLoop(devAgentCmd string, cfg Config) {
	// =======================================================
	// CORE LOOP: DEV 🔀 QA (SELF-HEALING)
	// =======================================================
	_ = os.Remove("workspace/qa_error.log")
	success := false

	for retry := 0; retry < MaxRetries; retry++ {
		// PHASE 1: DEVELOPMENT / REPAIR
		updateState(StageDev, retry)
		fmt.Printf("🤖 Activating %s CLI for Code Generation/Repair...\n", devAgentCmd)

		devPrompt, err := os.ReadFile(".agents/antigravity_dev_prompt.md")
		if err != nil {
			log.Fatalf("❌ Missing configuration file: .agents/antigravity_dev_prompt.md")
		}

		cmdDev := exec.Command(devAgentCmd,
			"--print", string(devPrompt),
			"--dangerously-skip-permissions",
			"--add-dir", "./workspace",
			"--add-dir", "./memory",
		)
		// Pass model via environment variable — agy reads ANTIGRAVITY_MODEL at startup
		if cfg.Dev.ModelName != "" {
			cmdDev.Env = append(os.Environ(), "ANTIGRAVITY_MODEL="+cfg.Dev.ModelName)
		}
		_ = cmdDev.Run()

		// PHASE 2: QA VERIFICATION (TEST SUITE)
		updateState(StageQA, retry)
		fmt.Println("🕵️ Running automated test verification: go test -v ./workspace/...")

		cmdQA := exec.Command("go", "test", "-v", "./workspace/...")
		var errQA bytes.Buffer
		cmdQA.Stdout = &errQA
		cmdQA.Stderr = &errQA

		if err := cmdQA.Run(); err != nil {
			fmt.Printf("⚠️ Tests failed on attempt %d! Writing to qa_error.log for AI self-healing...\n", retry+1)
			_ = os.WriteFile("workspace/qa_error.log", errQA.Bytes(), 0644)
			pipelineTelemetry.TotalRetriesUsed++
			time.Sleep(2 * time.Second)
		} else {
			fmt.Println("🎉 Excellent! 100% of the automated QA Test Suite passed.")
			_ = os.Remove("workspace/qa_error.log")
			if retry > 0 {
				pipelineTelemetry.CodeHealingSuccess = true
			}
			success = true
			break
		}
	}

	if !success {
		log.Fatalf("❌ [Harness Aborted] Agent attempted %d self-healing loops but failed QA. Manual intervention required!", MaxRetries)
	}

	// =======================================================
	// PHASE 3: HUMAN-IN-THE-LOOP (HITL INTERCEPTOR)
	// =======================================================
	updateState(StageHITL, 0)
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
	if finalInput != "y" && finalInput != "" { // Empty string (just Enter) or 'y' are treated as Yes
		fmt.Println("🛑 Disapproved! Terminating pipeline without updating memory.")
		os.Exit(1)
	}

	// =======================================================
	// PHASE 4: DEVOPS & LOCAL OUTCOME COMPILATION (DEVOPS AGENT)
	// =======================================================
	updateState(StageDevOps, 0)
	fmt.Printf("📝 Invoking local %s agent (%s) to construct deployment documentation...\n", cfg.DevOps.Agent, cfg.DevOps.ModelName)

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
		// Process ONLY the targeted subfolder for the current task
		featureFiles, _ := os.ReadDir(targetSubfolder)
		var allCode string
		for _, ff := range featureFiles {
			if !ff.IsDir() && strings.HasSuffix(ff.Name(), ".go") {
				content, _ := os.ReadFile(fmt.Sprintf("%s/%s", targetSubfolder, ff.Name()))
				allCode += string(content) + "\n"
			}
		}

		if allCode != "" {
			sysPrompt := fmt.Sprintf("You are a deployment release manager. Generate a short, bulleted markdown release note based on the provided Go code for the feature '%s'. Keep it brief.", parsedFeatureName)
			releaseNotes, errOllama := callOllama(cfg.DevOps.Agent, cfg.DevOps.ModelName, sysPrompt, allCode)
			notePath := fmt.Sprintf("%s/RELEASE_NOTES.md", targetSubfolder)
			if errOllama != nil {
				fmt.Printf("⚠️ Local Ollama communication failed for %s: %v\n", parsedFeatureName, errOllama)
			} else {
				_ = os.WriteFile(notePath, []byte(releaseNotes), 0644)
				fmt.Printf("📝 Generated %s automatically using local resources.\n", notePath)

				if ticketID != "" {
					if cfg.DevOps.MCPConfig != "" {
						fmt.Printf("🚀 Triggering dev_agent to update Linear ticket %s...\n", ticketID)
						linearPrompt := fmt.Sprintf("Tests passed and feature '%s' is complete. Please update the Linear ticket %s with the following release notes:\n%s", parsedFeatureName, ticketID, releaseNotes)

						// Note: The dev_agent (e.g., agy) does not natively support an isolated --config flag.
						// If MCP tools are required, they must be configured globally via the agent's CLI (e.g., 'agy mcp add').
						cmdLinear := exec.Command(devAgentCmd, "--print", linearPrompt)
						var outLinear bytes.Buffer
						cmdLinear.Stdout = &outLinear
						cmdLinear.Stderr = &outLinear
						if errLinear := cmdLinear.Run(); errLinear != nil {
							fmt.Printf("⚠️ Warning: Failed to update Linear ticket %s: %v\nOutput: %s\n", ticketID, errLinear, outLinear.String())
						} else {
							fmt.Printf("✅ Successfully updated Linear ticket %s.\n", ticketID)
						}
					} else {
						fmt.Println("ℹ️ Linear MCP config missing. Skipping Linear ticket update.")
					}
				} else {
					fmt.Println("ℹ️ No Ticket ID configured for this task. Skipping Linear ticket update.")
				}
			}
		}
	} else {
		// Fallback for legacy single-task mode: Iterate over the entire workspace
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

				sysPrompt := fmt.Sprintf("You are a deployment release manager. Generate a short, bulleted markdown release note based on the provided Go code for the feature '%s'. Keep it brief.", feature)
				releaseNotes, errOllama := callOllama(cfg.DevOps.Agent, cfg.DevOps.ModelName, sysPrompt, allCode)
				notePath := fmt.Sprintf("workspace/%s/RELEASE_NOTES.md", feature)
				if errOllama != nil {
					fmt.Printf("⚠️ Local Ollama communication failed for %s: %v\n", feature, errOllama)
				} else {
					_ = os.WriteFile(notePath, []byte(releaseNotes), 0644)
					fmt.Printf("📝 Generated %s automatically using local resources.\n", notePath)
					fmt.Println("ℹ️ Running in legacy workspace mode without explicit ticket ID. Skipping Linear ticket update.")
				}
			}
		}
	}

	// =======================================================
	// FINALIZE & MEMORY PROGRESSION
	// =======================================================
	updateState(StageCompact, 0)
	updateSystemMemory(cfg.DevOps.Agent, cfg.DevOps.ModelName)
	compactSystemMemory(cfg.DevOps.Agent, cfg.DevOps.ModelName)

	pipelineTelemetry.TotalDurationSeconds = time.Since(pipelineStartTime).Seconds()
	pipelineTelemetry.LinesOfCodeGenerated = countGeneratedLines()
	pipelineTelemetry.Timestamp = time.Now().Format(time.RFC3339)
	telemetryBytes, _ := json.MarshalIndent(pipelineTelemetry, "", "  ")
	_ = os.WriteFile("workspace/telemetry.json", telemetryBytes, 0644)

	updateState(StageDone, 0)
	fmt.Println("\n🎯 SPRINT PIPELINE RUN COMPLETE. Check your /workspace folder for final artifacts!")
}

// ExecuteBigEpic reads a directory of requirements, decomposes it into tasks, and runs the core loop for each task.
func ExecuteBigEpic(epicFolderPath, devAgentCmd string, cfg Config) {
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
	jsonPlan, err := callOllama(cfg.DevOps.Agent, cfg.DevOps.ModelName, sysPrompt, hugeContext.String())
	if err != nil {
		log.Fatalf("❌ Epic decomposition failed: %v", err)
	}

	var pipeline EpicPipeline
	// Try extracting JSON from backticks if the model returned markdown
	if strings.Contains(jsonPlan, "```json") {
		start := strings.Index(jsonPlan, "```json") + 7
		end := strings.LastIndex(jsonPlan, "```")
		if end > start {
			jsonPlan = jsonPlan[start:end]
		}
	}

	if err := json.Unmarshal([]byte(jsonPlan), &pipeline); err != nil {
		log.Fatalf("❌ Failed to parse epic JSON decomposition: %v\nRaw Output:\n%s", err, jsonPlan)
	}

	for i, task := range pipeline.SubTasks {
		fmt.Printf("\n🎬 [SPRINT %d/%d] Beginning implementation of Module: %s\n", i+1, len(pipeline.SubTasks), task.Name)

		dodContent := fmt.Sprintf("# TASK: %s\n- Target Subfolder: %s\n- Ticket ID: %s\n\n## Requirements\n%s",
			task.Name, task.TargetFolder, task.TicketID, task.PromptSpecs)
		_ = os.WriteFile("memory/definitions_of_done.md", []byte(dodContent), 0644)

		runCoreHarnessLoop(devAgentCmd, cfg)
	}

	fmt.Println("\n🏆 [EPIC COMPLETED] All files in the epic directory have been successfully implemented into modular packages!")
}

type BAConfig struct {
	Agent     string `json:"agent"`
	MCPConfig string `json:"mcp_config"`
}

type DevConfig struct {
	Agent     string `json:"agent"`
	ModelName string `json:"model_name"`
	MCPConfig string `json:"mcp_config"`
}

type DevOpsConfig struct {
	Agent     string `json:"agent"`
	ModelName string `json:"model_name"`
	MCPConfig string `json:"mcp_config"`
}

type Config struct {
	BA     BAConfig     `json:"ba"`
	Dev    DevConfig    `json:"dev"`
	DevOps DevOpsConfig `json:"devops"`
}

func main() {
	pipelineStartTime = time.Now()
	// Default configuration
	cfg := Config{
		BA: BAConfig{
			Agent:     "gemini",
			MCPConfig: ".mcp/ba_notion.json",
		},
		Dev: DevConfig{
			Agent:     "agy",
			ModelName: "gemini-2.5-flash",
			MCPConfig: "",
		},
		DevOps: DevOpsConfig{
			Agent:     "ollama",
			ModelName: "hermes3:8b",
			MCPConfig: ".mcp/devops_linear.json",
		},
	}

	file, err := os.ReadFile("harness_config.json")
	if err == nil {
		json.Unmarshal(file, &cfg)
	}

	taskFlag := flag.String("task", "", "Raw requirement to trigger Phase 0 Business Analyst")
	epicFlag := flag.String("epic", "", "Path to a directory containing epic requirements for decomposition")
	baAgentCmd := flag.String("ba-agent", cfg.BA.Agent, "Command/binary to execute for Phase 0 Business Analyst")
	devAgentCmd := flag.String("dev-agent", cfg.Dev.Agent, "Command/binary to execute for Phase 1 Developer Coding")
	devAgentModel := flag.String("dev-model", cfg.Dev.ModelName, "Model name for the Dev agent (sets ANTIGRAVITY_MODEL env var)")
	devOpsAgent := flag.String("devops-agent", cfg.DevOps.Agent, "CLI agent to execute for Phase 3 DevOps documentation (e.g., ollama)")
	devOpsModel := flag.String("devops-model", cfg.DevOps.ModelName, "Model name to execute for Phase 3 DevOps documentation")
	flag.Parse()

	fmt.Println("🚀 ACTIVATING GO HARNESS PIPELINE v2026.1")
	_ = os.MkdirAll("memory", 0755)
	_ = os.MkdirAll("workspace", 0755)

	fmt.Printf("⚙️  Configuration:\n")
	fmt.Printf("   - BA Agent:    %s\n", *baAgentCmd)
	fmt.Printf("   - Dev Agent:   %s (model: %s)\n", *devAgentCmd, *devAgentModel)
	fmt.Printf("   - DevOps Agent: %s\n", *devOpsAgent)
	fmt.Printf("   - DevOps Model: %s\n", *devOpsModel)

	if *epicFlag != "" {
		cfg.Dev.ModelName = *devAgentModel
		ExecuteBigEpic(*epicFlag, *devAgentCmd, cfg)
		return
	}

	if *taskFlag != "" {
		fmt.Printf("\n🎯 Raw requirement received: '%s'\n", *taskFlag)
		fmt.Printf("🤖 BA Agent (%s) is drafting the Definitions of Done...\n", *baAgentCmd)

		baPrompt := fmt.Sprintf(`
You are an expert Business Analyst. 
Take this raw requirement: "%s".
Analyze it and generate a standardized, highly technical 'definitions_of_done.md' layout.
Output ONLY the strict markdown checklist content. Do not include any chat filler or explanations.
`, *taskFlag)

		baArgs := []string{"run", baPrompt}
		// Note: The gemini CLI does not natively support an isolated --config flag for MCP.
		// MCP servers should be added globally via 'gemini mcp add'.
		if cfg.BA.MCPConfig != "" {
			if _, err := os.Stat(cfg.BA.MCPConfig); err == nil {
				fmt.Printf("ℹ️  Found BA MCP config at %s, but passing it dynamically is not supported by the agent CLI.\n", cfg.BA.MCPConfig)
			}
		}

		cmdBA := exec.Command(*baAgentCmd, baArgs...)
		var outBA, errBA bytes.Buffer
		cmdBA.Stdout = &outBA
		cmdBA.Stderr = &errBA

		if err := cmdBA.Run(); err != nil {
			log.Fatalf("❌ BA Agent (%s) failed: %v \n%s", *baAgentCmd, err, errBA.String())
		}

		_ = os.WriteFile("memory/definitions_of_done.md", outBA.Bytes(), 0644)
		fmt.Println("✅ Successfully generated memory/definitions_of_done.md.")
	}

	cfg.Dev.ModelName = *devAgentModel
	runCoreHarnessLoop(*devAgentCmd, cfg)
}
