package docs

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// GenerateDocumentation spins up a local llama-server in the background,
// waits for it to become healthy, and runs openwiki --update to sync documentation.
func GenerateDocumentation(modelName string) error {
	fmt.Printf("📚 [OpenWiki] Starting auto-documentation phase...\n")

	// Determine model path, assuming standard HuggingFace cache format logic used in other agents
	// Here we just pass the modelName to llama-server, which can handle -hf repos natively
	// Note: We use 8192 context size to give openwiki enough room to read files
	serverCmd := exec.Command("llama-server",
		"-hf", modelName,
		"--port", "8080",
		"-c", "8192",
		"--parallel", "1",
		"--cont-batching",
	)
	
	// Hide server output to avoid cluttering pipeline logs, or redirect to a log file
	logFile, _ := os.Create("workspace/openwiki_server.log")
	if logFile != nil {
		serverCmd.Stdout = logFile
		serverCmd.Stderr = logFile
		defer logFile.Close()
	}

	if err := serverCmd.Start(); err != nil {
		return fmt.Errorf("failed to start llama-server: %v", err)
	}

	// Ensure the server process is killed when this function exits
	defer func() {
		fmt.Printf("📚 [OpenWiki] Shutting down local llama-server...\n")
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
			serverCmd.Wait()
		}
	}()

	// Wait for the server to be ready
	fmt.Printf("📚 [OpenWiki] Waiting for local OpenAI-compatible endpoint to become ready (max 60s)...\n")
	ready := false
	for i := 0; i < 30; i++ {
		resp, err := http.Get("http://localhost:8080/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				ready = true
				break
			}
		}
		time.Sleep(2 * time.Second)
	}

	if !ready {
		return fmt.Errorf("llama-server failed to become ready within 60s")
	}

	fmt.Printf("📚 [OpenWiki] Server ready. Running 'openwiki --update'...\n")

	// Run openwiki
	wikiCmd := exec.Command("openwiki", "--update")
	
	// Set the environment variables exactly as openwiki expects for OpenAI compatibility
	wikiCmd.Env = append(os.Environ(),
		"OPENWIKI_PROVIDER=openai-compatible",
		"OPENAI_COMPATIBLE_BASE_URL=http://localhost:8080/v1",
		"OPENAI_COMPATIBLE_API_KEY=dummy-key-for-local",
		fmt.Sprintf("OPENWIKI_MODEL_ID=%s", modelName),
	)

	wikiCmd.Stdout = os.Stdout
	wikiCmd.Stderr = os.Stderr

	if err := wikiCmd.Run(); err != nil {
		return fmt.Errorf("openwiki --update failed: %v", err)
	}

	fmt.Printf("✅ [OpenWiki] Documentation successfully updated!\n")
	return nil
}
