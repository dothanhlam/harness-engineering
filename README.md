# Harness Orchestration Engine & Validation Modules (v2026.1)

Welcome to the **Harness Orchestration System**, a robust, state-aware automation pipeline and high-performance validation engine engineered for Go ecosystems. This project integrates autonomous AI agents, automated quality assurance workflows, and local LLM orchestration to streamline development from initial analysis through deployment delivery.

---

## 🚀 Key Features

### 1. 🔄 Multi-Stage Orchestration Pipeline (`internal/pipeline/`)

```mermaid
flowchart TD
    BA["0. BA STAGE (ollama) - Read PRD -> Write memory/DoD"]
    DEV["1. DEV_CODING (ollama) - Generate code into subfolder"]
    QA["2. QA_TESTING (go test) - Parallel Audit & Test Suite - Auto-heal up to 3 times"]
    BA_REF["3. BA_REFACTOR - Delegation Protocol (Rewrite DoD)"]
    HITL["4. HUMAN_IN_THE_LOOP - Manual terminal approval"]
    DEVOPS["5. DEVOPS_DELIVER - Ollama Release Notes"]
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
*   **`DEV_CODING`**: Invokes the configured developer agent (`agy` CLI by default) to synthesize and self-verify project files. In Epic mode, this can run concurrently across isolated workspaces.
*   **`QA_TESTING`**: Runs in parallel using goroutines:
    *   **Security Audit**: Strictly analyzes generated `.go` files for forbidden imports (like `os/exec`), destructive commands, or hardcoded credentials.
    *   **Test Suite**: Automatically executes the repository's test hooks (`go test -v ./workspace/...`). 
    If QA fails, combined errors are logged to `workspace/qa_error.log` for AI self-healing.
*   **`BA_REFACTOR` (Delegation Protocol)**: A dynamic non-linear delegation loop. If the Developer agent exhausts its QA healing retries, the orchestrator safely delegates back to the BA agent to rewrite and clarify the `definitions_of_done.md` based on the compilation errors.
*   **`HUMAN_IN_THE_LOOP`**: Halts the pipeline, requiring user approval via terminal (auto-approves after 30s) before integration.
*   **`DEVOPS_DELIVER`**: Calls a local Ollama instance to summarize the codebase changes and compile `workspace/RELEASE_NOTES.md`.
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

| Flag | Default Value | Description |
|---|---|---|
| `-task` | `""` | Raw requirement string. Triggers Phase 0 Business Analyst to update `definitions_of_done.md` |
| `-epic` | `""` | Path to a directory containing epic requirements. Triggers the Epic Orchestrator. |
| `-parallel-epic` | `false` | Run epic sub-tasks concurrently with isolated memory workspaces. |
| `-ba-agent` | `"ollama"` | Binary/CLI name used for Phase 0 Business Analyst |
| `-ba-model` | `"hermes3:8b"` | Model name for the Phase 0 Business Analyst agent |
| `-dev-agent` | `"ollama"` | Binary/CLI name used for Phase 1 Developer Coding |
| `-dev-model` | `"gemma4:e4b"` | Model name for the Dev agent |
| `-devops-agent`| `"ollama"`| Binary/CLI name used for Phase 3 DevOps documentation |
| `-devops-model`| `"hermes3:8b"`| Model name to execute for Phase 3 DevOps documentation |

**Example usages:**
```bash
# Run with standard agents, triggering the BA phase with a raw task requirement
go run main.go -task "Create a secure bcrypt hashing module"

# Trigger the Epic Orchestrator to decompose and implement a large folder of requirements concurrently
go run main.go -epic "./requirements/auth_epic/" -parallel-epic

# Switch the dev coding agent to claude (if testing alternative models)
go run main.go -dev-agent claude -dev-model claude-sonnet-4-20250514
```

**Developer Agent Invocation:**
The Developer phase leverages `ollama` for local LLM inference. In CLI mode it runs as a subprocess; in Docker mode it communicates via HTTP API:
```bash
# Local (CLI subprocess)
ollama run gemma4:e4b "$DEV_PROMPT" --verbose

# Docker (HTTP API via OLLAMA_HOST env)
POST http://ollama:11434/api/generate {"model": "gemma4:e4b", "prompt": "...", "stream": true}
```

---

## 📁 Repository Structure

```
harness-app/
├── .agents/
│   └── antigravity_dev_prompt.md  # Autonomous Developer Agent configuration
├── internal/                      # Modular Harness Orchestrator core
│   ├── agent/                     # Pluggable CLI/HTTP agent adapter
│   ├── config/                    # JSON Configuration loader
│   ├── memory/                    # System blueprint & AI compaction logic
│   ├── pipeline/                  # Core loops (epic, sequential, parallel)
│   ├── qa/                        # Concurrent security audit & test runner
│   └── telemetry/                 # Mutex-protected execution metrics
├── memory/
│   ├── definitions_of_done.md    # Product specifications & validation criteria
│   └── lessons_learned.md        # Debugging guidelines & operational history
├── scripts/
│   └── docker-entrypoint.sh      # Docker entrypoint (wait for Ollama, pull models)
├── workspace/                    # Core development artifacts
│   ├── email_validation/         # Modular package: Email Validation
│   ├── landing_page/             # Modular package: Landing Page
│   ├── password/                 # Modular package: Bcrypt Hashing
│   ├── random/                   # Modular package: Random Generation
│   └── state.json                # JSON active pipeline stage tracker
├── harness_config.json           # Agent and Model configurations
├── Dockerfile                    # Multi-stage Go build
├── docker-compose.yml            # Harness + Ollama sidecar
├── go.mod                        # Module definition (github.com/dothanhlam/harness-app)
├── main.go                       # Slim orchestrator entrypoint
└── README.md                     # Project documentation (this file)
```

---

## 🛠️ Getting Started & Usage

### Prerequisites

| Requirement | Local Dev | Docker |
|---|:---:|:---:|
| **Go** 1.26.1+ | ✅ Required | ❌ Not needed |
| **Ollama** (running locally) | ✅ Required | ❌ Not needed |
| **Docker & Docker Compose** | ❌ Not needed | ✅ Required |

---

### Option A: Run Locally

Requires Go and Ollama installed on your machine:

```bash
# 1. Pull the required models
ollama pull hermes3:8b
ollama pull gemma4:e4b

# 2. Build the binary
make build

# 3. Run with a task
./harness_bin --task "Create a secure bcrypt hashing module"

# 4. Run an epic concurrently
./harness_bin --epic "./requirements/auth_epic/" --parallel-epic
```

---

### Option B: Run with Docker (Recommended)

No local Go or Ollama installation needed — everything runs in containers.

#### Quick Start

```bash
# Build the Docker images
make docker-build

# Run a single task (models are auto-pulled on first run)
make docker-run TASK="Create a hello world Go program"

# Stop everything
make docker-down
```

#### Architecture

The Docker setup uses a **sidecar architecture** with two containers:

```
┌─────────────────────────────────────────────────────────────┐
│  Docker Compose Network                                     │
│                                                             │
│  ┌──────────────────┐     HTTP API      ┌────────────────┐  │
│  │  harness-pipeline│ ──────────────►   │ harness-ollama │  │
│  │  (Go binary)     │  :11434/api/      │ (LLM server)   │  │
│  └────────┬─────────┘  generate         └────────┬───────┘  │
│           │                                      │          │
└───────────┼──────────────────────────────────────┼──────────┘
            │ bind mount                           │ named volume
            ▼                                      ▼
   ./workspace/  (host)                   ollama_models (docker)
   ./memory/     (host)                   (~5GB model weights)
```

| Container | Image | Purpose |
|---|---|---|
| `harness-ollama` | `ollama/ollama:latest` | Serves LLM inference on port `11434` |
| `harness-pipeline` | Built from `Dockerfile` | Runs the Go orchestrator, talks to Ollama via HTTP |

#### 📂 Volume Mounts — Accessing Generated Code on Your Host

The `workspace/` and `memory/` directories are **bind-mounted** from your host machine into the container. This means:

- **All code generated by the AI agents inside Docker appears instantly on your host filesystem.**
- You can open `./workspace/` in your IDE (VS Code, GoLand, etc.) and watch files appear in real-time as the pipeline runs.
- Pipeline state (`workspace/state.json`) and telemetry (`workspace/telemetry.json`) are also visible on the host.

```yaml
# From docker-compose.yml — these lines make it work:
volumes:
  - ./workspace:/app/workspace   # ← Generated code lives here on your host
  - ./memory:/app/memory         # ← Agent memory (DoD, blueprint) on your host
  - ./harness_config.json:/app/harness_config.json:ro  # ← Config (read-only)
```

> **Tip:** After a pipeline run, browse `./workspace/<feature_name>/` on your host to see the generated Go packages, tests, and release notes.

#### Model Storage

Model weights (`hermes3:8b` ~4.7GB, `gemma4:e4b`) are stored in a persistent **Docker named volume** (`harness-ollama-models`). They are only downloaded on first run and survive container restarts.

```bash
# Check which models are cached
curl http://localhost:11434/api/tags | jq '.models[].name'

# Force re-pull a model
docker compose exec ollama ollama pull hermes3:8b
```

#### All Docker Commands

```bash
make docker-build   # Build the harness image
make docker-up      # Start stack in detached mode (ollama + harness)
make docker-run TASK="your requirement"  # Run a single task
make docker-down    # Stop and remove containers

# Useful docker compose commands
docker compose logs -f              # Follow all output
docker compose logs -f harness      # Follow harness output only
docker compose exec ollama ollama list  # List cached models
```

---

### Running the Test Suite

```bash
go test -v ./workspace/...
```

**Example output:**
```text
=== RUN   TestHashPassword
--- PASS: TestHashPassword (0.26s)
=== RUN   TestIsValidEmail
--- PASS: TestIsValidEmail (0.00s)
=== RUN   TestHandler_Index
--- PASS: TestHandler_Index (0.00s)
...
PASS
ok  	github.com/dothanhlam/harness-app/workspace/password	2.734s
ok  	github.com/dothanhlam/harness-app/workspace/landing_page	0.968s
```
