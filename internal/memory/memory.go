package memory

import (
	"fmt"
	"os"

	"github.com/dothanhlam/harness-app/internal/agent"
)

// UpdateSystemMemory progressively analyzes architectural correlations and archives features to Mem0.
func UpdateSystemMemory(mem0Client *Mem0Client, devopsAgent *agent.AgentSpec) {
	fmt.Println("🧠 [PROGRESSIVE MEMORY] Analyzing modular architectural correlations...")
	currentDoD, err := os.ReadFile("memory/definitions_of_done.md")
	if err != nil {
		fmt.Printf("⚠️ Could not read definitions_of_done.md: %v\n", err)
		return
	}

	sysPrompt := `You are an Enterprise System Architect. Analyze the new requirement against the system context.
Identify structural dependencies, package reusability, or architectural correlations. Return ONLY the concise markdown log.`

	userPrompt := fmt.Sprintf("=== REQS ===\n%s", string(currentDoD))
	fullPrompt := fmt.Sprintf("SYSTEM INSTRUCTIONS:\n%s\n\nUSER INPUT:\n%s", sysPrompt, userPrompt)
	
	correlations, err := devopsAgent.Execute(fullPrompt)
	if err == nil {
		memoryContent := fmt.Sprintf("Architectural correlation for feature:\n%s\n\nDetails:\n%s", string(currentDoD), correlations)
		err = mem0Client.AddMemory(memoryContent)
		if err != nil {
			fmt.Printf("⚠️ Failed to store memory in Mem0: %v\n", err)
		} else {
			fmt.Println("✅ [PROGRESSIVE MEMORY] System architecture map synchronized to Mem0.")
		}
	} else {
		fmt.Printf("⚠️ DevOps agent failed to execute: %v\n", err)
	}
}
