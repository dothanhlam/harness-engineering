package memory

import (
	"fmt"
	"os"
	"time"

	"github.com/dothanhlam/harness-app/internal/agent"
)

// UpdateSystemMemory progressively analyzes architectural correlations and archives features.
func UpdateSystemMemory(devopsAgent *agent.AgentSpec) {
	fmt.Println("🧠 [PROGRESSIVE MEMORY] Analyzing modular architectural correlations...")
	oldBlueprint, _ := os.ReadFile("memory/system_blueprint.md")
	currentDoD, _ := os.ReadFile("memory/definitions_of_done.md")

	sysPrompt := `You are an Enterprise System Architect. Analyze the new requirement against the existing blueprint.
Identify structural dependencies, package reusability, or architectural correlations. Return ONLY the concise markdown log.`

	userPrompt := fmt.Sprintf("=== REQS ===\n%s\n=== BLUEPRINT ===\n%s", string(currentDoD), string(oldBlueprint))
	fullPrompt := fmt.Sprintf("SYSTEM INSTRUCTIONS:\n%s\n\nUSER INPUT:\n%s", sysPrompt, userPrompt)
	correlations, err := devopsAgent.Execute(fullPrompt)
	if err == nil {
		newContent := fmt.Sprintf("%s\n\n## [ARCHIVED FEATURE - %s]\n%s", string(oldBlueprint), time.Now().Format("2006-01-02 15:04"), correlations)
		_ = os.WriteFile("memory/system_blueprint.md", []byte(newContent), 0644)
		fmt.Println("✅ [PROGRESSIVE MEMORY] System architecture map synchronized.")
	}
}

// CompactSystemMemory uses AI to optimize the system_blueprint.md if it grows too large.
func CompactSystemMemory(devopsAgent *agent.AgentSpec) {
	blueprintPath := "memory/system_blueprint.md"
	data, err := os.ReadFile(blueprintPath)
	if err != nil || len(data) < 3000 {
		return
	}

	fmt.Println("🧹 [MEMORY COMPACTION] System memory is getting too long. Activating AI optimization...")
	sysPrompt := `You are an Enterprise System Architect. The system blueprint memory is getting too long. 
Review the entire text and compress older historical feature logs into a single unified 'Current System Architecture Map' section. 
Keep the latest 2 features fully intact, but summarize all previous ones into architectural line items. Return ONLY the new compact markdown.
Be extremely concise. Return bullet points only. Limit your response to under 150 words. Do not write filler structural prose.`

	fullPrompt := fmt.Sprintf("SYSTEM INSTRUCTIONS:\n%s\n\nUSER INPUT:\n%s", sysPrompt, string(data))
	compacted, err := devopsAgent.Execute(fullPrompt)
	if err == nil {
		_ = os.WriteFile(blueprintPath, []byte(compacted), 0644)
		fmt.Println("✅ [MEMORY COMPACTION] Successfully optimized Memory context window!")
	}
}
