# Harness Orchestration Engine & Validation Modules (v2026.1)

[![Go](https://img.shields.io/github/go-mod/go-version/dothanhlam/harness-engineering?logo=go&logoColor=white&label=Go)](go.mod)
[![Version](https://img.shields.io/github/v/tag/dothanhlam/harness-engineering?label=version&color=blue)](https://github.com/dothanhlam/harness-engineering/tags)
[![Last commit](https://img.shields.io/github/last-commit/dothanhlam/harness-engineering)](https://github.com/dothanhlam/harness-engineering/commits)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Inference](https://img.shields.io/badge/inference-llama.cpp-orange)](https://github.com/ggml-org/llama.cpp)
[![Docker](https://img.shields.io/badge/docker-ready-2496ED?logo=docker&logoColor=white)](Dockerfile)

Welcome to the **Harness Orchestration System**, a robust, state-aware automation pipeline and high-performance validation engine engineered for Go ecosystems. This project integrates autonomous AI agents, automated quality assurance workflows, and local LLM orchestration to streamline development from initial analysis through deployment delivery.

---

## 🚀 Key Features

### 1. 🔄 Multi-Stage Orchestration Pipeline (`internal/pipeline/`)

```mermaid
flowchart TD
    BA["0. BA STAGE (llama.cpp) - Read PRD -> Write memory/DoD"]
    DEV["1. DEV_CODING (llama.cpp) - Generate code into subfolder"]
    QA["2. QA_TESTING (go test) - Parallel Audit & Test Suite - Auto-heal up to 3 times"]
    BA_REF["3. BA_REFACTOR - Delegation Protocol (Rewrite DoD)"]
    HITL["4. HUMAN_IN_THE_LOOP - Manual terminal approval"]
    DEVOPS["5. DEVOPS_DELIVER - llama.cpp Release Notes"]
    COMPACT["6. MEMORY_COMPACTION - Mem0 Archiving"]
    DONE["7. COMPLETED"]

    BA --> DEV
    DEV --> QA
    QA -- "Delegation Loop (Fail)" --> BA_REF
    BA_REF --> DEV
    QA -- "Pass" --> HITL
    HITL -- "Approve" --> DEVOPS
    DEVOPS --> COMPACT
    COMPACT --> DONE
```

The orchestrator transitions autonomously through defined pipeline states (`internal/pipeline/stages.go`), persisting its current state to `workspace/state.json`. It features robust **Goroutine Concurrency** and **Mutex-protected Telemetry Tracking** to export runtime metrics to `workspace/telemetry.json`.
*   **`DEV_CODING`**: Invokes the configured developer agent (`agy` CLI by default) to synthesize and self-verify project files. The output target subfolder is dynamically determined by parsing the `# TASK:` or `- Target Subfolder:` tags generated in the `definitions_of_done.md`. If absent, it gracefully defaults to `workspace/default_task/`. In Epic mode, this runs concurrently across isolated workspaces.
*   **`QA_TESTING`**: Runs in parallel using goroutines:
    *   **Security Audit**: Strictly analyzes generated `.go` files for forbidden imports (like `os/exec`), destructive commands, or hardcoded credentials.
    *   **Test Suite**: Automatically executes the repository's test hooks (`go test -v ./workspace/...`). 
    If QA fails, combined errors are logged to `workspace/qa_error.log` for AI self-healing.
*   **`BA_REFACTOR` (Delegation Protocol)**: A dynamic non-linear delegation loop. If the Developer agent exhausts its QA healing retries, the orchestrator safely delegates back to the BA agent to rewrite and clarify the `definitions_of_done.md` based on the compilation errors.
*   **`HUMAN_IN_THE_LOOP`**: Halts the pipeline, requiring user approval via terminal (auto-approves after 30s) before integration.
*   **`DEVOPS_DELIVER`**: Calls a local llama.cpp instance to summarize the codebase changes and compile `workspace/RELEASE_NOTES.md`.
*   **`MEMORY_COMPACTION`**: Progressively analyzes requirements and archives architectural correlations directly into the local Mem0 vector database for semantic search.
*   **`COMPLETED`**: Finalizes the build, exports pipeline telemetry, and closes the loop.

### 2. 🛡️ Security & Validation Modules (`workspace/`)
A modular approach containing highly secure and robust validation components:
*   **Password Hashing**: Implements bcrypt hashing with strict constraints (72-byte limit, minimum cost factor of 10) to mitigate common vulnerabilities. Utilizes zero-allocation techniques (`unsafe.Slice`) for high-performance memory safety.
*   **Email Validation**: Comprehensive unit testing suite for validating email structures and edge cases.
*   **Landing Page**: A self-contained, modular package that serves a highly premium, glassmorphic marketing and technical landing page featuring interactive pipeline animations, and an asynchronous secure inquiry form with full server-side validation.

---

## ⚙️ Configuration & Agent Switching

You can switch the agents, models, and endpoints used in each phase dynamically using `harness_config.json` at the root of the project, or via CLI flags which override the defaults:

All agent/model flags default to `""`, meaning "use whatever `harness_config.json` specifies". Pass one only to override the config file for a single run.

| Flag | Default Value | Description |
|---|---|---|
| `-project-dir` | `"."` | Target project directory the pipeline operates on |
| `-task` | `""` | Raw requirement string. Triggers Phase 0 Business Analyst to update `definitions_of_done.md` |
| `-target` | `""` | Explicit target subfolder to output generated code (e.g. `workspace/custom_folder`) |
| `-epic` | `""` | Path to a directory containing epic requirements. Triggers the Epic Orchestrator. |
| `-parallel-epic` | `false` | Run epic sub-tasks concurrently with isolated memory workspaces. |
| `-force-regression` | `false` | Run QA across the whole workspace instead of only the new subfolder |
| `-ba-agent` | `""` (config) | Binary/CLI name used for Phase 0 Business Analyst |
| `-ba-model` | `""` (config) | Model path for the Phase 0 Business Analyst agent |
| `-dev-agent` | `""` (config) | Binary/CLI name used for Phase 1 Developer Coding |
| `-dev-model` | `""` (config) | Model path for the Dev agent |
| `-devops-agent`| `""` (config) | Binary/CLI name used for Phase 3 DevOps documentation |
| `-devops-model`| `""` (config) | Model path to execute for Phase 3 DevOps documentation |

**Example usages:**
```bash
# Run with standard agents, triggering the BA phase with a raw task requirement
go run main.go -task "Create a secure bcrypt hashing module"

# Force output to a specific subfolder instead of generating one dynamically
go run main.go -task "Create a secure bcrypt hashing module" -target "workspace/security"

# Trigger the Epic Orchestrator to decompose and implement a large folder of requirements concurrently
go run main.go -epic "./requirements/auth_epic/" -parallel-epic

# Switch the dev coding agent to claude (if testing alternative models)
go run main.go -dev-agent claude -dev-model claude-sonnet-4-20250514
```

**Engine Migration: Ollama to llama.cpp**
The system has migrated its core execution engine from Ollama to the native `llama.cpp` flat binary (`llama-completion`) for local LLM inference (while retaining HTTP fallback for Docker). This provides zero-overhead execution and enables direct low-level parameter tuning.

**Installing New Models for llama.cpp:**
llama.cpp natively supports downloading and caching models from the Hugging Face Hub using the `hf://` prefix. Models will be stored in `~/.cache/huggingface/hub`.
1. Update `harness_config.json` with the path to the new model using the format `hf://<repo>:<quant>`:
   ```json
   "dev": {
     "agent": "llama_cpp",
     "model_name": "hf://bartowski/Qwen2.5-Coder-7B-Instruct-GGUF:Q4_K_M"
   }
   ```
   A `model_name` without the `hf://` prefix is treated as a local filesystem path, and a leading `~/` is expanded.

**Developer Agent Invocation (Local CLI):**
```bash
# Local (CLI subprocess via llama.cpp)
llama-completion -hf bartowski/Qwen2.5-Coder-7B-Instruct-GGUF:Q4_K_M -c 16384 --flash-attn on --no-display-prompt -p "$DEV_PROMPT"
```

### 🧠 Sizing the Dev Model to Your Host

Local inference is **memory-bound, not CPU-bound**. A model that does not fit does not fail cleanly — it thrashes, taking the host down with it. Budget roughly `weights + KV cache`, where the KV cache scales linearly with `context_window`:

| Dev model | Weights | KV @ 16k ctx | Total | Minimum host RAM |
|---|---|---|---|---|
| Qwen2.5-Coder-7B Q4_K_M *(default)* | ~4.7 GB | ~0.9 GB | ~6 GB | 16 GB |
| Qwen2.5-Coder-14B Q4_K_M | ~8.4 GB | ~3.0 GB | ~12 GB | 32 GB |

On Apple Silicon the practical ceiling is the GPU wired limit, which defaults to roughly 2/3 of physical RAM (~10.6 GB on a 16 GB machine) — not total RAM. Exceeding it will hang the machine, not just the run. Halving `context_window` roughly halves the KV cache if you need to fit a larger model.

Two guardrails exist for when this goes wrong anyway:
*   **Dev agent timeout** (`internal/pipeline/stages.go`): a llama.cpp dev invocation is capped at 10 minutes, so a wedged model load surfaces as an actionable error instead of an indefinite hang.
*   **Epic memory guard** (`internal/pipeline/epic.go`): in `-parallel-epic` mode, local dev agents are serialized to one at a time — each concurrent agent would otherwise load its own full copy of the weights. CLI agents (Claude et al.) are network-bound and still fan out freely.

---

## 📁 Repository Structure

```
harness-engineering/
├── .agents/
│   └── antigravity_dev_prompt.md  # Autonomous Developer Agent configuration
├── internal/                      # Modular Harness Orchestrator core
│   ├── agent/                     # Pluggable CLI/HTTP agent adapter
│   ├── config/                    # JSON Configuration loader
│   ├── docs/                      # OpenWiki auto-documentation generator
│   ├── events/                    # JSONL event bus, tailer & run records
│   ├── memory/                    # System blueprint & AI compaction logic
│   ├── monitor/                   # Terminal & browser dashboards over the event stream
│   ├── pipeline/                  # Core loops (epic, sequential, parallel)
│   ├── qa/                        # Concurrent security audit & test runner
│   └── telemetry/                 # Mutex-protected execution metrics
├── memory/
│   ├── definitions_of_done.md    # Product specifications & validation criteria
│   └── lessons_learned.md        # Debugging guidelines & operational history
├── scripts/
│   └── docker-entrypoint.sh      # Docker entrypoint
├── workspace/                    # Core development artifacts
│   ├── email_validation/         # Modular package: Email Validation
│   ├── landing_page/             # Modular package: Landing Page
│   ├── password/                 # Modular package: Bcrypt Hashing
│   ├── random/                   # Modular package: Random Generation
│   ├── events.jsonl              # Append-only run event stream (truncated per run)
│   └── state.json                # JSON active pipeline stage tracker
├── harness_config.json           # Agent and Model configurations
├── Dockerfile                    # Multi-stage Go build
├── docker-compose.yml            # Harness sidecar
├── go.mod                        # Module definition (github.com/dothanhlam/harness-engineering)
├── main.go                       # Slim orchestrator entrypoint
├── LICENSE                       # MIT License
└── README.md                     # Project documentation (this file)
```

---

## 🛠️ Integration & Usage Guidelines

The Harness Engine is a standalone framework designed to integrate into **any project directory**, whether it's a fresh scaffolding or an existing repository.

### Method 1: Git Submodule (Recommended for Monorepos)

If you are managing a large project or monorepo, you can include the framework as a git submodule.

```bash
cd /path/to/my-project

# Add the framework as a submodule
git submodule add https://github.com/dothanhlam/harness-engineering.git .harness-framework

# Build the framework
cd .harness-framework
go build -o harness .
cd ..

# Initialize Harness in your project
./.harness-framework/harness init --project-dir .

# Run the pipeline on your project
./.harness-framework/harness run --project-dir . --task "Create a secure bcrypt hashing module"
```

### Method 2: Global Binary (Local Execution)

Clone the framework independently and run the binary globally.

```bash
# Clone and build globally
git clone https://github.com/dothanhlam/harness-engineering.git
cd harness-engineering
go build -o harness .
sudo mv harness /usr/local/bin/

# Navigate to your actual project
cd /path/to/my-project

# Initialize and run
harness init --project-dir .
harness run --project-dir . --task "Add a Subtract function to the existing math_utils module"
```

### Method 3: Docker (No Local Setup)

You can run the framework completely via Docker without installing Go or `llama.cpp`. The `--project-dir` will be mounted inside the container.

```bash
cd /path/to/my-project

# 1. Clone the framework to build the image (one-time setup)
git clone https://github.com/dothanhlam/harness-engineering.git .harness-framework
cd .harness-framework
docker build -t harness-pipeline .
cd ..

# 2. Initialize Harness in your project using Docker
docker run --rm \
  -v $(pwd):/app/project \
  harness-pipeline init --project-dir /app/project

# 3. Run the pipeline (Mount your local AI models cache)
docker run --rm \
  -v $(pwd):/app/project \
  -v ~/.cache/huggingface/hub:/root/.cache/huggingface/hub \
  harness-pipeline run --project-dir /app/project --task "Create a hello world Go program"
```

> **Note:** The pipeline dynamically detects the project's ecosystem (Go, Node.js `package.json`, Python `pytest.ini` or `requirements.txt`) to run the appropriate QA test commands!

---

## 👁️ Monitoring a Run (`harness monitor`)

A run is headless and owns stdin for the HITL gate, so monitoring is a **separate process** that tails the append-only event stream at `workspace/events.jsonl`. Start it before, during, or after a run — a finished log simply replays.

```bash
# Terminal 1: start the run
harness run --project-dir . --task "Create a secure bcrypt hashing module"

# Terminal 2: follow it live
harness monitor --project-dir .

# Or serve a browser dashboard instead (localhost only)
harness monitor --project-dir . --web --port 7777

# Replay a finished run and exit, without agent output noise
harness monitor --project-dir . --once --no-output
```

| Flag | Default | Description |
|---|---|---|
| `-project-dir` | `"."` | Project directory whose run to monitor |
| `-path` | `""` | Event log path (defaults to `<project-dir>/workspace/events.jsonl`) |
| `-once` | `false` | Replay the current log and exit instead of following live |
| `-no-output` | `false` | Hide mirrored agent output for a high-level stage view |
| `-color` | `"auto"` | Colorize output: `auto`\|`always`\|`never` |
| `-web` | `false` | Serve a browser dashboard instead of tailing the terminal |
| `-port` | `7777` | Port for `--web` (bound to localhost only) |

Monitoring is strictly non-fatal: if the event log cannot be opened the run continues without it, and a monitor process can never take the pipeline down.

---

### Step-by-Step Breakdown

**1. Project Initialization (`harness init`)**
When you run `init`, the framework scaffolds the following in your target directory:
- `.harness/rules.json`: Dynamic QA rules and security audit configurations (e.g., banning `rm -rf`).
- `.agents/`: Autonomous agent instructions and prompts (e.g., instructing the Dev agent to output `SEARCH/REPLACE` blocks).

**2. Running the Pipeline (`harness run`)**
When running on an existing project, the Developer Agent utilizes a Surgical File Editing technique (Aider-style `<<<< ==== >>>>` blocks). This ensures it safely patches existing code instead of blindly overwriting entire files.

---

---

### Running Tests on the Target Project

The Harness pipeline automatically executes tests during the `QA_TESTING` phase by dynamically detecting the target project's ecosystem (e.g., `npm run test` for Node.js, `pytest` for Python, `go test` for Go). 

To manually run the test suite on your target Go project after generation:

```bash
cd /path/to/my-project
go test -v ./...
```

**Example output:**
```text
=== RUN   TestHashPassword
--- PASS: TestHashPassword (0.26s)
=== RUN   TestIsValidEmail
--- PASS: TestIsValidEmail (0.00s)
...
PASS
ok  	github.com/my-org/my-project/password	2.734s
```

---

## 📄 License

Released under the [MIT License](LICENSE). Copyright (c) 2026 Do Thanh Lam.
