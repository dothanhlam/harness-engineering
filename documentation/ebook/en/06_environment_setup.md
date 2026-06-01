# Chapter 6: Environment Setup & Tooling

To run the Harness Engineering pipeline successfully on your local machine, you need to install and configure the necessary foundation of tools. The orchestrator (`main.go`) acts as the conductor, but it relies on these external CLI tools to execute the heavy lifting.

## 1. Go (Golang)
The orchestrator and the generated code modules are written in Go. You will need **Go 1.21 or higher**.
* **Mac/Linux**: 
  ```bash
  brew install go
  ```
* **Windows/Other**: Download the installer from the [official Go website](https://go.dev/dl/).

## 2. The BA Agent: Gemini CLI
We use the `gemini` command-line tool as our Business Analyst (Phase 0). It is responsible for decomposing raw tasks into the strict checklists found in `definitions_of_done.md`.
* Ensure you have the Gemini CLI installed and authenticated on your machine so the `gemini run` commands execute seamlessly.

## 3. The Developer Agent: Antigravity (`agy`)
The Developer agent (Phase 1) is powered by the `agy` CLI (Antigravity). This is an autonomous agent that reads our system prompts and writes the code inside the `workspace/` folder.
* Install the `agy` CLI per your internal organizational tools.
* The orchestrator automatically passes the `--dangerously-skip-permissions` flag to allow `agy` to run autonomously without pausing for file write permissions, though it is strictly sandboxed to the `workspace/` directory via the `--add-dir` flags.

## 4. The DevOps Agent: Ollama
For Phase 3 (Release Notes generation and Memory Compaction), we use localized LLMs to save on cloud API costs and ensure complete privacy for our source code. 
* Download and install **Ollama** from [ollama.com](https://ollama.com/).
* Once installed, pull the model we use for documentation generation (configured in `harness_config.json`):
  ```bash
  ollama pull hermes3:8b
  ```
* Ensure the Ollama server is running in the background before you start the harness:
  ```bash
  ollama serve
  ```

## 5. Verify the Setup
Once you have Go, Gemini, Agy, and Ollama installed, you can verify your setup by running a simple test task:
```bash
go run main.go -task "Create a simple hello world module"
```

If the pipeline successfully flows through BA -> DEV -> QA -> AUDIT -> HITL -> DEVOPS without throwing any "command not found" errors, your environment is perfectly configured!
