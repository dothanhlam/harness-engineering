package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dothanhlam/harness-engineering/internal/agent"
	"github.com/dothanhlam/harness-engineering/internal/telemetry"
)

// parseDoDTarget extracts the target subfolder and feature name from a
// definitions_of_done.md body. It falls back to workspace/default_task when the
// DoD omits both the "- Target Subfolder:" and "# TASK:" markers.
func parseDoDTarget(dod string) (targetSubfolder, featureName string) {
	for _, line := range strings.Split(dod, "\n") {
		if strings.HasPrefix(line, "- Target Subfolder: ") {
			targetSubfolder = strings.TrimSpace(strings.TrimPrefix(line, "- Target Subfolder: "))
			featureName = filepath.Base(targetSubfolder)
		} else if strings.HasPrefix(line, "# TASK: ") && featureName == "" {
			featureName = strings.TrimSpace(strings.TrimPrefix(line, "# TASK: "))
		}
	}
	if featureName == "" {
		featureName = "default_task"
	}
	if targetSubfolder == "" {
		targetSubfolder = fmt.Sprintf("workspace/%s", featureName)
	}
	return targetSubfolder, featureName
}

// buildLlamaDevPrompt assembles the full prompt for a local llama.cpp dev agent.
// Local models have no filesystem tools, so the DoD, blueprint, and any QA error
// log must be injected inline. The "feature_name"/"filename" placeholders in the
// prompt template are substituted with the parsed feature name.
func buildLlamaDevPrompt(template, featureName, dod, blueprint, qaLog string) string {
	var sb strings.Builder
	if featureName != "" {
		template = strings.ReplaceAll(template, "feature_name", featureName)
		template = strings.ReplaceAll(template, "filename", featureName)
	}
	sb.WriteString(template)
	if dod != "" {
		sb.WriteString("\n\n=== CONTEXT: memory/definitions_of_done.md ===\n")
		sb.WriteString(dod)
	}
	if blueprint != "" {
		sb.WriteString("\n\n=== CONTEXT: memory/system_blueprint.md ===\n")
		sb.WriteString(blueprint)
	}
	if qaLog != "" {
		sb.WriteString("\n\n=== CONTEXT: workspace/qa_error.log (Fix these errors!) ===\n")
		sb.WriteString(qaLog)
	}
	return sb.String()
}

// readGoSources concatenates all non-test .go files in a feature folder into a
// single string for summarization. Returns "" when there is nothing to read.
func readGoSources(targetFolder string) string {
	entries, err := os.ReadDir(targetFolder)
	if err != nil {
		return ""
	}
	var all strings.Builder
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(targetFolder, e.Name()))
		if err == nil {
			all.Write(content)
			all.WriteString("\n")
		}
	}
	return all.String()
}

// generateReleaseNotes summarizes the Go code in a feature folder into
// RELEASE_NOTES.md using the DevOps agent. llama.cpp runs are bounded by a
// timeout with a graceful fallback so a slow local model never blocks delivery.
// It is a no-op when the folder has no Go sources.
func generateReleaseNotes(devops *agent.AgentSpec, targetFolder, featureName string, tracker *telemetry.Tracker) {
	allCode := readGoSources(targetFolder)
	if allCode == "" {
		return
	}

	sysPrompt := fmt.Sprintf("You are a deployment release manager. Generate a short, bulleted markdown release note based on the provided Go code for the feature '%s'. Keep it brief. Be extremely concise. Return bullet points only. Limit your response to under 150 words. Do not write filler structural prose.", featureName)
	fullPrompt := fmt.Sprintf("SYSTEM INSTRUCTIONS:\n%s\n\nUSER INPUT:\n%s", sysPrompt, allCode)

	var releaseNotes string
	var err error
	var usage agent.TokenUsage
	if devops.Agent == "llama_cpp" {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		releaseNotes, usage, err = devops.ExecuteWithContext(ctx, fullPrompt)
		tracker.AddTokens(usage.PromptTokens, usage.EvalTokens)
		if err != nil {
			fmt.Printf("⚠️ [%s THERMAL THROTTLING] DevOps agent timed out. Gracefully falling back to save CPU cycles...\n", strings.ToUpper(devops.Agent))
			releaseNotes = "- DevOps auto-generation aborted (thermal fallback).\n- Check commits for details."
			err = nil
		}
	} else {
		releaseNotes, usage, err = devops.Execute(fullPrompt)
		tracker.AddTokens(usage.PromptTokens, usage.EvalTokens)
	}

	if err != nil {
		fmt.Printf("⚠️ DevOps Agent communication failed for %s: %v\n", featureName, err)
		return
	}
	notePath := filepath.Join(targetFolder, "RELEASE_NOTES.md")
	_ = os.WriteFile(notePath, []byte(releaseNotes), 0644)
	fmt.Printf("📝 Generated %s automatically using local resources.\n", notePath)
}
