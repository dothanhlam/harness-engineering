# Chapter 4: Daily Workflow for Engineers

How do you interact with the Harness on a daily basis? You don't need to be an expert in how the internal orchestration loops work. You just need to know how to prompt it via the command line.

## Configuration

We use a combination of `harness_config.json` and CLI flags to control which AI agents are actively working. By default, **all three phases run fully local via llama.cpp** — no cloud calls, complete privacy for our source code:
* **Business Analyst (Phase 0)** uses `llama_cpp` with `Hermes-3-Llama-3.1-8B`.
* **Developer (Phase 1)** uses `llama_cpp` with `Qwen2.5-Coder-14B`.
* **DevOps (Phase 3)** uses `llama_cpp` with `Hermes-3-Llama-3.1-8B`.

Every agent is swappable per phase. To try a different model, edit `harness_config.json` or pass a flag (e.g. `-dev-model hf://bartowski/Qwen2.5-Coder-14B-Instruct-GGUF:Q4_K_M`). You can even swap in a cloud CLI agent such as Claude — the repo ships an inactive `_dev_claude_backup` block as a template.

*Optional: MCP integrations.* The `.mcp/` folder retains optional Model Context Protocol templates (`ba_notion.json`, `devops_linear.json`) for CLI agents that support it — e.g. reading PRDs from Notion or updating Linear tickets. These are **not** wired into the default local llama.cpp agents; enable them only when you switch to a CLI agent that speaks MCP.

## The two subcommands: `init` and `run`

* `harness init -project-dir .` scaffolds `.harness/rules.json` and `.agents/` into a target project (see Chapter 6 and the Integration guide).
* `harness run …` executes the pipeline.

When developing from source you can run it directly with `go run main.go run …` (the older `go run main.go -task …` form still works via a backward-compatible fallback to `run`).

## Running a Task

If you have a specific feature you want the system to build, pass it as a raw string using the `-task` flag:

```bash
go run main.go run -task "Create a highly efficient Fibonacci function in Go with O(n) complexity"
```

The system will start from the BA phase, draft the requirements, and build the code in the `workspace/` folder.

## Running an Epic

If you have a folder full of raw markdown requirement files, you can use the Epic Orchestrator to decompose them into decoupled sub-features:

```bash
# Sequential — each sub-feature runs the full BA→Dev→QA→HITL loop, one at a time
go run main.go run -epic "./requirements/v2_launch/"

# Parallel — Dev + QA fan out across sub-features with isolated memory per task
go run main.go run -epic "./requirements/v2_launch/" -parallel-epic
```

See the **Integration & Epics guide** (`documentation/INTEGRATION.md`) for how to structure the requirement files.

## Where is the output?

* **`memory/`**: This is where the AI stores its context. You will find `definitions_of_done.md` (the checklist) and `system_blueprint.md` (the architectural map).
* **`workspace/`**: This is where the actual Go code is generated. Each feature gets its own clean subfolder, alongside `state.json` (the live pipeline stage) and `RELEASE_NOTES.md`.
* **`telemetry.json`** (project root): Check this file to see how long the pipeline took, how many lines of code were generated, and how many self-healing retries were used.

Welcome to the future of engineering!
