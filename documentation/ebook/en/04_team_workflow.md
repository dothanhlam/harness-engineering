# Chapter 4: Daily Workflow for Engineers

How do you interact with the Harness on a daily basis? You don't need to be an expert in how the internal orchestration loops work. You just need to know how to prompt it via the command line.

## Configuration

We use a combination of `harness_config.json` and CLI flags to control which AI agents are actively working. 
By default:
* **Business Analyst (Phase 0)** uses the `gemini` CLI (with Notion MCP capabilities).
* **Developer (Phase 1)** uses the `agy` CLI (Antigravity).
* **DevOps (Phase 3)** uses a local `ollama` instance (with Linear MCP capabilities).

*Note: MCP (Model Context Protocol) allows the AI agents to reach out to external tools like Notion to read your PRDs and Linear to update your tickets automatically.*

## Running a Task

If you have a specific feature you want the system to build, pass it as a raw string using the `-task` flag:

```bash
go run main.go -task "Create a highly efficient Fibonacci function in Go with O(n) complexity"
```

The system will start from Phase 0, draft the requirements, and build the code in the `workspace/` folder.

## Running an Epic

If you are a Product Manager and have a folder full of raw markdown requirement files, you can use the Epic Orchestrator:

```bash
go run main.go -epic "./requirements/v2_launch/"
```

The system will decompose all files in the folder into decoupled sub-features and process them one by one.

## Where is the output?

* **`memory/`**: This is where the AI stores its context. You will find `definitions_of_done.md` (the checklist) and the Mem0 vector database backend (submodule).
* **`workspace/`**: This is where the actual Go code is generated. Each feature gets its own clean subfolder.
* **`workspace/telemetry.json`**: Check this file to see how long the pipeline took, how many lines of code were generated, and how many self-healing retries were used.

Welcome to the future of engineering!
