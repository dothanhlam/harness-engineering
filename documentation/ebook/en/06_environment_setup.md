# Chapter 6: Environment Setup & Tooling

To run the Harness Engineering pipeline successfully on your local machine, you need to install and configure the necessary foundation of tools. The orchestrator (`main.go` + `internal/`) acts as the conductor, but it relies on the `llama.cpp` runtime to execute the heavy lifting.

## 1. Go (Golang)
The orchestrator and the generated code modules are written in Go. You will need **Go 1.26 or higher** (see `go.mod`).
* **Mac/Linux**:
  ```bash
  brew install go
  ```
* **Windows/Other**: Download the installer from the [official Go website](https://go.dev/dl/).

## 2. The Inference Runtime: llama.cpp
All three agents (BA, Developer, DevOps) run locally by default through **llama.cpp**. The harness invokes the `llama-completion` binary, so it must be on your `PATH`.
* **Mac/Linux**:
  ```bash
  brew install llama.cpp
  ```
* **Other platforms**: build from source per the [llama.cpp repository](https://github.com/ggml-org/llama.cpp).

Running locally means zero cloud API cost and complete privacy for our source code.

## 3. Models (auto-downloaded from Hugging Face)
llama.cpp natively downloads and caches GGUF models from the Hugging Face Hub using the `hf://` prefix — you don't need to fetch them manually. Models are stored under `~/.cache/huggingface/hub`.

The defaults in `harness_config.json` are:
```json
"ba":     { "agent": "llama_cpp", "model_name": "hf://NousResearch/Hermes-3-Llama-3.1-8B-GGUF:Q4_K_M" },
"dev":    { "agent": "llama_cpp", "model_name": "hf://bartowski/Qwen2.5-Coder-14B-Instruct-GGUF:Q4_K_M" },
"devops": { "agent": "llama_cpp", "model_name": "hf://NousResearch/Hermes-3-Llama-3.1-8B-GGUF:Q4_K_M" }
```
The first run of each model will download it (this can take a while); subsequent runs use the cache. You can also point `model_name` at a local `.gguf` file path instead of an `hf://` URL.

*Optional — swapping in a cloud CLI agent.* If you'd rather use a CLI agent such as Claude for a phase, copy the inactive `_dev_claude_backup` block in `harness_config.json` over the relevant key and make sure that CLI is installed and authenticated. MCP integrations under `.mcp/` (Notion, Linear) apply only to such CLI agents.

## 4. Sandbox & Skill Management
To keep development organized and maintain modular agent contexts, the pipeline operates inside isolated directories (`workspace/`, `memory/`, `.agents/`).

You can initialize these directories and prepare your local skill toolkit using the Harness Makefile:

### Sandbox Initialization
First, run the initialization recipe to create the required directories and instantiate a baseline config:
```bash
make init
```

### Interactive Skill Provisioning
Next, run the interactive installer to choose which expert domain skills to load:
```bash
make skills
```
This script checks for Node and NPX, then prompts you with a selection of expert packages (e.g., ClickHouse patterns, TDD guidelines, or Go Clean Architecture).

You can also list your installed skills and prune them when no longer needed:
```bash
# List all currently installed skills
make list-skills

# Remove a specific skill folder
make remove-skill SKILL=<skill-folder-name>
```

## 5. Verify the Setup
Once the environment is configured, verify the entire setup with a trivial task:
```bash
go run main.go run -task "Create a simple hello world module"
```

If the pipeline flows through BA → DEV_CODING → QA_TESTING → HUMAN_IN_THE_LOOP → DEVOPS_DELIVER → MEMORY_COMPACTION → COMPLETED without any "command not found" errors, your environment is perfectly configured!
