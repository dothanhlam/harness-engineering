# Harness Orchestration Engine & Validation Modules (v2026.1)

Welcome to the **Harness Orchestration System**, a robust, state-aware automation pipeline and high-performance validation engine engineered for Go ecosystems. This project integrates autonomous AI agents, automated quality assurance workflows, and local LLM orchestration to streamline development from initial analysis through deployment delivery.

---

## 🚀 Key Features

### 1. 🔄 Multi-Stage Orchestration Pipeline (`main.go` at root)

```mermaid
flowchart TD
    BA["1. BA STAGE (Gemini)<br>Read PRD -> Write memory/DoD"]
    DEV["2. DEV STAGE (agy)<br>Generate code into subfolder"]
    QA["3. QA STAGE (go test)<br>Auto-heal up to 3 times"]
    AUDIT["4. GOVERNANCE & AUDIT GATE<br>Scan for malicious code & check security rules"]
    HITL["5. HUMAN-IN-THE-LOOP<br>Manual terminal approval"]
    DEVOPS["6. DEVOPS & MEMORY COMPACTION<br>Compress blueprint -> Linear MCP Update -> Export telemetry.json"]

    BA --> DEV
    DEV --> QA
    QA -- "Delegation Loop (Fail)" --> BA
    QA -- "Pass" --> AUDIT
    AUDIT -- "Pass" --> HITL
    HITL -- "Approve" --> DEVOPS
```

The main orchestrator transitions autonomously through defined pipeline states, persisting its current state to `workspace/state.json`. It now features advanced **Telemetry Tracking** to export runtime metrics (like generated LOC, timestamps, and self-healing success rates) to `workspace/telemetry.json`.
*   **`DEV_CODING`**: Invokes the configured developer agent (`agy` CLI by default) to synthesize and self-verify project files.
*   **`SECURITY_AUDIT`**: An integrated guardrail that intercepts execution before QA to strictly analyze generated `.go` files for forbidden imports (like `os/exec`), destructive commands, or hardcoded credentials. It fails the build immediately on violation to initiate a safe healing cycle.
*   **`QA_TESTING`**: Automatically executes the repository's test hooks (`go test -v ./workspace/...`). If tests fail, errors are logged to `workspace/qa_error.log` for AI self-healing.
*   **`BA_REFACTOR` (Delegation Protocol)**: A dynamic non-linear delegation loop. If the Developer agent exhausts its QA healing retries, the orchestrator safely delegates back to the BA agent (Gemini) to rewrite and clarify the `definitions_of_done.md` based on the compilation errors.
*   **`DEVOPS_DELIVER`**: Calls a local Ollama instance running a configurable model to summarize the codebase changes and compile `workspace/RELEASE_NOTES.md`.
*   **`COMPLETED`**: Finalizes the build, exports pipeline telemetry, and closes the loop.

### 2. 🛡️ Security & Validation Modules (`workspace/`)
A modular approach containing highly secure and robust validation components:
*   **Password Hashing**: Implements bcrypt hashing with strict constraints (72-byte limit, minimum cost factor of 10) to mitigate common vulnerabilities. Utilizes zero-allocation techniques (`unsafe.Slice`) for high-performance memory safety.
*   **Email Validation**: Comprehensive unit testing suite for validating email structures and edge cases.
*   **Landing Page**: A self-contained, modular package that serves a highly premium, glassmorphic marketing and technical landing page featuring interactive pipeline animations, and an asynchronous secure inquiry form with full server-side validation.


---

## ⚙️ Configuration & Agent Switching

You can switch the agents, models, and endpoints used in each phase dynamically using `config.json` at the root of the project, or via CLI flags which override the defaults:

| Flag | Default Value | Description |
|---|---|---|
| `-task` | `""` | Raw requirement string. Triggers Phase 0 Business Analyst to update `definitions_of_done.md` |
| `-epic` | `""` | Path to a directory containing epic requirements. Triggers the Epic Orchestrator for bulk decomposition and implementation. |
| `-ba-agent` | `"gemini"` | Binary/CLI name used for Phase 0 Business Analyst |
| `-dev-agent` | `"agy"` | Binary/CLI name used for Phase 1 Developer Coding |
| `-devops-model` | `"hermes3:8b"`| Local Ollama model used for Phase 3 Release Notes |
| `-devops-url` | `"http://localhost:11434/api/chat"` | Local Ollama HTTP API endpoint |

**Example usages:**
```bash
# Run with standard agents, triggering the BA phase with a raw task requirement
go run main.go -task "Create a secure bcrypt hashing module"

# Trigger the Epic Orchestrator to decompose and implement a large folder of requirements
go run main.go -epic "./requirements/auth_epic/"

# Switch the dev coding agent to another CLI tool (e.g. customized-coder)
go run main.go -dev-agent customized-coder

# Switch the DevOps documentation model to llama3
go run main.go -devops-model llama3
```

**Developer Agent Invocation:**
The Developer phase leverages the `agy` CLI for autonomous execution, passing goals and granting restricted access:
```bash
agy --print "$DEV_PROMPT" --dangerously-skip-permissions --add-dir ./workspace --add-dir ./memory
```

---

## 📁 Repository Structure

```
harness-app/
├── .agents/
│   └── antigravity_dev_prompt.md  # Autonomous Developer Agent configuration
├── memory/
│   ├── definitions_of_done.md    # Product specifications & validation criteria
│   ├── lessons_learned.md        # Debugging guidelines & operational history
│   └── system_blueprint.md       # Auto-syncing modular feature architectures
├── workspace/                    # Core development artifacts
│   ├── email_validation/         # Modular package: Email Validation
│   ├── landing_page/             # Modular package: Landing Page
│   ├── password/                 # Modular package: Bcrypt Hashing
│   ├── random/                   # Modular package: Random Generation
│   └── state.json                # JSON active pipeline stage tracker
├── config.json                   # Agent and Model default configurations
├── go.mod                        # Module definition (github.com/dothanhlam/harness-app)
├── main.go                       # Main Go Harness pipeline orchestrator
└── README.md                     # Project documentation (this file)
```

---

## 🛠️ Getting Started & Usage

### Prerequisites
*   **Go** (1.21 or higher)
*   **Ollama**: Must be installed and running locally (`ollama serve`) with the configured model (e.g., `hermes3:8b`) to execute the automated DevOps documentation phase.
*   **agy CLI**: The Antigravity autonomous developer agent must be installed to execute code generation.

---

### Running the Test Suite
The project contains comprehensive unit tests that cover all modular implementations within the `workspace/` folder. Run them using:

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
