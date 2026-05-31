# Harness Orchestration Engine & Validation Modules (v2026.1)

Welcome to the **Harness Orchestration System**, a robust, state-aware automation pipeline and high-performance validation engine engineered for Go ecosystems. This project integrates autonomous AI agents, automated quality assurance workflows, and local LLM orchestration to streamline development from initial analysis through deployment delivery.

---

## 🚀 Key Features

### 1. 🔄 Multi-Stage Orchestration Pipeline (`main.go` at root)
The main orchestrator transitions autonomously through defined pipeline states, persisting its current state to `workspace/state.json`:
*   **`DEV_CODING`**: Invokes the configured developer agent (`agy` CLI by default) to synthesize and self-verify project files.
*   **`QA_TESTING`**: Automatically executes the repository's test hooks (`go test -v ./workspace/...`). If tests fail, errors are logged to `workspace/qa_error.log`.
*   **`DEVOPS_DELIVER`**: Calls a local Ollama instance running a configurable model to summarize the codebase changes and compile `workspace/RELEASE_NOTES.md`.
*   **`COMPLETED`**: Finalizes the build and closes the loop.

### 2. 🛡️ Security & Validation Modules (`workspace/`)
A modular approach containing highly secure and robust validation components:
*   **Password Hashing**: Implements bcrypt hashing with strict constraints (72-byte limit, minimum cost factor of 10) to mitigate common vulnerabilities. Utilizes zero-allocation techniques (`unsafe.Slice`) for high-performance memory safety.
*   **Email Validation**: Comprehensive unit testing suite for validating email structures and edge cases.
*   **Landing Page**: A self-contained, modular package that serves a highly premium, glassmorphic marketing and technical landing page featuring interactive pipeline animations, and an asynchronous secure inquiry form with full server-side validation.


---

## ⚙️ Configuration & Agent Switching

You can switch the agents, models, and endpoints used in each phase dynamically using CLI flags:

| Flag | Default Value | Description |
|---|---|---|
| `-task` | `""` | Raw requirement string. Triggers Phase 0 Business Analyst to update `definitions_of_done.md` |
| `-ba-agent` | `"gemini"` | Binary/CLI name used for Phase 0 Business Analyst |
| `-dev-agent` | `"agy"` | Binary/CLI name used for Phase 1 Developer Coding |
| `-devops-model` | `"hermes3:8b"`| Local Ollama model used for Phase 3 Release Notes |
| `-devops-url` | `"http://localhost:11434/api/chat"` | Local Ollama HTTP API endpoint |

**Example usages:**
```bash
# Run with standard agents, triggering the BA phase with a raw task requirement
go run main.go -task "Create a secure bcrypt hashing module"

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
harness-engineering/
├── .agents/
│   └── antigravity_dev_prompt.md  # Autonomous Developer Agent configuration
├── memory/
│   ├── definitions_of_done.md    # Product specifications & validation criteria
│   └── lessons_learned.md        # Debugging guidelines & operational history
├── workspace/                    # Core development artifacts
│   ├── go.mod                    # Module definition (harness-engineering)
│   ├── main.go                   # Implementation (e.g. Bcrypt Hashing)
│   ├── main_test.go              # Unit test suite (e.g. Email validation, Hashing)
│   └── state.json                # JSON active pipeline stage tracker
├── main.go                       # Main Go Harness pipeline orchestrator
└── README.md                     # Project documentation (this file)
```

---

## 🛠️ Getting Started & Usage

### Prerequisites
*   **Go** (1.21 or higher)
*   *(Optional)* **Ollama** running locally with the configured model to execute the automated DevOps documentation phase.
*   *(Optional)* **agy** / **Gemini CLI** installed in your environment for autonomous code generation.

---

### Running the Test Suite
The project contains comprehensive unit tests that cover module implementations (like Email Validation and Bcrypt constraints). Run them using:

```bash
cd workspace
go test -v ./...
```

**Example output:**
```text
=== RUN   TestHashPassword_Success
--- PASS: TestHashPassword_Success (0.12s)
=== RUN   TestHashPassword_LengthLimit
--- PASS: TestHashPassword_LengthLimit (0.00s)
=== RUN   TestComparePassword_Success
--- PASS: TestComparePassword_Success (0.11s)
=== RUN   TestComparePassword_Failure
--- PASS: TestComparePassword_Failure (0.12s)
=== RUN   TestIsValidEmail_Valid
--- PASS: TestIsValidEmail_Valid (0.00s)
...
PASS
ok  	harness-engineering/workspace	0.360s
```
