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
* Ensure you have the Gemini CLI installed and authenticated on your machine so the `gemini run` commands execute seamlessly. It can optionally leverage an MCP configuration (`.mcp/ba_notion.json`) for Notion integration.

## 3. The Developer Agent: Antigravity (`agy`)
The Developer agent (Phase 1) is powered by the `agy` CLI (Antigravity). This is an autonomous agent that reads our system prompts and writes the code inside the `workspace/` folder.
* Install the `agy` CLI per your internal organizational tools.
* The orchestrator automatically passes the `--dangerously-skip-permissions` flag to allow `agy` to run autonomously without pausing for file write permissions, though it is strictly sandboxed to the `workspace/` and `memory/` directories via the `--add-dir` flags. It also dynamically sets the `ANTIGRAVITY_MODEL` environment variable (e.g., to `gemini-2.5-flash`).

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
* When integrated with Linear via an MCP configuration (`.mcp/devops_linear.json`), this phase can automatically update tickets with the generated release notes.

## 5. Sandbox & Skill Management
To keep development organized and maintain modular agent contexts, the pipeline operates inside isolated directories (`workspace/`, `memory/`, `.agents/skills/`). 

You can initialize these directories and prepare your local skill toolkit using the Harness Makefile:

### Sandbox Initialization
First, run the initialization recipe to create the required directories and instantiate a baseline config:
```bash
make init
```

### Interactive Skill Provisioning
Next, run the interactive installer to choose which expert domain skills from the awesome-skills catalog to load:
```bash
make skills
```
This script will check for Node and NPX, and prompt you with a selection of expert packages (e.g., ClickHouse patterns, TDD guidelines, or Go Clean Architecture).

You can also list your installed skills and prune them when no longer needed:
```bash
# List all currently installed skills
make list-skills

# Remove a specific skill folder
make remove-skill SKILL=<skill-folder-name>
```

## 6. Verify the Setup
Once you have the environment configured and the required skills provisioned, you can verify the entire setup:
```bash
make run -- -task "Create a simple hello world module"
```

If the pipeline successfully flows through BA -> DEV -> QA (Audit & Tests) -> HITL -> DEVOPS -> MEMORY COMPACT without throwing any "command not found" errors, your environment is perfectly configured!
