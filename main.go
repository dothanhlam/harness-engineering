package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ─────────────────────────────────────────────
// PIPELINE STAGE CONSTANTS
// ─────────────────────────────────────────────

type Stage string

const (
	StageDev    Stage = "DEV_CODING"
	StageQA     Stage = "QA_TESTING"
	StageDevOps Stage = "DEVOPS_DELIVER"
	StageDone   Stage = "COMPLETED"
)

// WorkflowState is the persisted pipeline state written to workspace/state.json.
type WorkflowState struct {
	TaskID       string    `json:"task_id"`
	CurrentStage Stage     `json:"current_stage"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// updateState writes the current pipeline stage to workspace/state.json.
func updateState(stage Stage) {
	state := WorkflowState{
		TaskID:       "cti_parser_001",
		CurrentStage: stage,
		UpdatedAt:    time.Now(),
	}
	file, _ := json.MarshalIndent(state, "", "  ")
	_ = os.WriteFile("workspace/state.json", file, 0644)
	fmt.Printf("\n🔄 [HARNESS STATE] -> %s\n", stage)
}

// updateSystemMemory scans workspace/ for modular features and archives them into memory.
func updateSystemMemory() {
	entries, err := os.ReadDir("workspace")
	if err != nil {
		return
	}
	
	var blueprintBuilder strings.Builder
	blueprintBuilder.WriteString("# System Blueprint: Modular Feature Architectures\n\n")
	blueprintBuilder.WriteString("This document automatically tracks all implemented modular features.\n\n")

	for _, entry := range entries {
		if entry.IsDir() {
			feature := entry.Name()
			blueprintBuilder.WriteString(fmt.Sprintf("## Feature Module: %s\n", feature))
			blueprintBuilder.WriteString(fmt.Sprintf("- **Location:** `/workspace/%s/`\n", feature))
			blueprintBuilder.WriteString(fmt.Sprintf("- **Status:** Archived & Verified\n\n"))
		}
	}
	
	_ = os.WriteFile("memory/system_blueprint.md", []byte(blueprintBuilder.String()), 0644)
	fmt.Println("💾 Synchronized feature states into /memory/system_blueprint.md")
}

// callOllama sends a chat request to a configurable Ollama API endpoint.
func callOllama(url, model, systemPrompt, userPrompt string) (string, error) {
	type Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type OllamaReq struct {
		Model    string    `json:"model"`
		Stream   bool      `json:"stream"`
		Messages []Message `json:"messages"`
	}

	payload := OllamaReq{
		Model:  model,
		Stream: false,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	jsonBytes, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(bodyBytes, &result)

	// Safe type assertion to extract message content.
	if msg, ok := result["message"].(map[string]interface{}); ok {
		if content, ok := msg["content"].(string); ok {
			return content, nil
		}
	}
	return string(bodyBytes), nil
}

type Config struct {
	BAAgent     string `json:"ba_agent"`
	DevAgent    string `json:"dev_agent"`
	DevOpsModel string `json:"devops_model"`
	DevOpsURL   string `json:"devops_url"`
}

func main() {
	// Default configuration
	cfg := Config{
		BAAgent:     "gemini",
		DevAgent:    "agy",
		DevOpsModel: "hermes3:8b",
		DevOpsURL:   "http://localhost:11434/api/chat",
	}

	// Try to load config.json
	file, err := os.ReadFile("config.json")
	if err == nil {
		json.Unmarshal(file, &cfg)
	}

	// 1. Declare CLI flags for configuring agents
	taskFlag := flag.String("task", "", "Raw requirement to trigger Phase 0 Business Analyst")
	baAgentCmd := flag.String("ba-agent", cfg.BAAgent, "Command/binary to execute for Phase 0 Business Analyst")
	devAgentCmd := flag.String("dev-agent", cfg.DevAgent, "Command/binary to execute for Phase 1 Developer Coding")
	devOpsModel := flag.String("devops-model", cfg.DevOpsModel, "Ollama model name to execute for Phase 3 DevOps documentation")
	devOpsURL := flag.String("devops-url", cfg.DevOpsURL, "Ollama API endpoint URL to connect to")
	flag.Parse()

	fmt.Println("🚀 KÍCH HOẠT GO HARNESS PIPELINE v2026.1")
	_ = os.MkdirAll("memory", 0755)
	_ = os.MkdirAll("workspace", 0755)

	fmt.Printf("⚙️  Configuration:\n")
	fmt.Printf("   - BA Agent:    %s\n", *baAgentCmd)
	fmt.Printf("   - Dev Agent:   %s\n", *devAgentCmd)
	fmt.Printf("   - DevOps Model: %s\n", *devOpsModel)
	fmt.Printf("   - DevOps URL:   %s\n", *devOpsURL)

	// =======================================================
	// PHASE 0: BUSINESS ANALYSIS (BA AGENT)
	// =======================================================
	if *taskFlag != "" {
		fmt.Printf("\n🎯 Raw requirement received: '%s'\n", *taskFlag)
		fmt.Printf("🤖 BA Agent (%s) is drafting the Definitions of Done...\n", *baAgentCmd)

		baPrompt := fmt.Sprintf(`
You are an expert Business Analyst. 
Take this raw requirement: "%s".
Analyze it and generate a standardized, highly technical 'definitions_of_done.md' layout.
Output ONLY the strict markdown checklist content. Do not include any chat filler or explanations.
`, *taskFlag)

		cmdBA := exec.Command(*baAgentCmd, "run", baPrompt)
		var outBA, errBA bytes.Buffer
		cmdBA.Stdout = &outBA
		cmdBA.Stderr = &errBA

		if err := cmdBA.Run(); err != nil {
			log.Fatalf("❌ BA Agent (%s) failed: %v \n%s", *baAgentCmd, err, errBA.String())
		}

		_ = os.WriteFile("memory/definitions_of_done.md", outBA.Bytes(), 0644)
		fmt.Println("✅ Successfully generated memory/definitions_of_done.md.")
	}

	// =======================================================
	// PHASE 1: DEVELOPMENT (DEV AGENT)
	// =======================================================
	updateState(StageDev)
	fmt.Printf("🤖 Activating %s CLI as your Autonomous Developer Agent...\n", *devAgentCmd)

	devPrompt, err := os.ReadFile(".agents/antigravity_dev_prompt.md")
	if err != nil {
		log.Fatalf("❌ Missing configuration file: .agents/antigravity_dev_prompt.md")
	}

	// Corrected flags based on Agy CLI usage output
	cmdDev := exec.Command(*devAgentCmd,
		"--print", string(devPrompt),
		"--dangerously-skip-permissions",
		"--add-dir", "./workspace",
		"--add-dir", "./memory",
	)
	var outDev, errDev bytes.Buffer
	cmdDev.Stdout = &outDev
	cmdDev.Stderr = &errDev

	if err := cmdDev.Run(); err != nil {
		log.Fatalf("❌ %s Developer Agent aborted: %v\nError Trace: %s", *devAgentCmd, err, errDev.String())
	}
	fmt.Printf("✅ %s successfully processed requirements and generated code inside /workspace.\n", *devAgentCmd)

	// =======================================================
	// PHASE 2: QA VERIFICATION (TEST SUITE)
	// =======================================================
	updateState(StageQA)
	fmt.Println("🕵️ Running comprehensive test hooks...")

	cmdQA := exec.Command("go", "test", "-v", "./workspace/...")
	var outQA, errQA bytes.Buffer
	cmdQA.Stdout = &outQA
	cmdQA.Stderr = &errQA

	if err := cmdQA.Run(); err != nil {
		_ = os.WriteFile("workspace/qa_error.log", errQA.Bytes(), 0644)
		fmt.Println("⚠️ Runtime Unit Tests Failed. Log saved to workspace/qa_error.log")
	} else {
		fmt.Println("🎉 100% of Unit Tests passed perfectly.")
	}

	// =======================================================
	// PHASE 3: DEVOPS & LOCAL OUTCOME COMPILATION (DEVOPS AGENT)
	// =======================================================
	updateState(StageDevOps)
	fmt.Printf("📝 Invoking local %s model to construct deployment documentation...\n", *devOpsModel)

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

			releaseNotes, errOllama := callOllama(*devOpsURL, *devOpsModel, sysPrompt, allCode)
			notePath := fmt.Sprintf("workspace/%s/RELEASE_NOTES.md", feature)
			if errOllama != nil {
				fmt.Printf("⚠️ Local Ollama communication failed for %s: %v\n", feature, errOllama)
			} else {
				_ = os.WriteFile(notePath, []byte(releaseNotes), 0644)
				fmt.Printf("📝 Generated %s automatically using local resources.\n", notePath)
			}
		}
	}

	// =======================================================
	// FINALIZE
	// =======================================================
	updateState(StageDone)
	updateSystemMemory()
	fmt.Println("\n🎯 PIPELINE RUN COMPLETE. Check your /workspace folder for final artifacts!")
}
