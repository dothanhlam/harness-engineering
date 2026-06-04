# Chapter 2: The Core Pipeline Architecture

At the heart of our repository is `main.go`. This is the orchestrator—the brain of the Harness System. It acts as the manager for our AI agents.

## The Multi-Stage Orchestration Pipeline

Our pipeline is broken down into 5 distinct stages. When you trigger the harness, it moves through these stages autonomously.

```mermaid
flowchart TD
    BA["1. BA STAGE (Gemini)<br>Read PRD -> Write memory/DoD"]
    DEV["2. DEV STAGE (agy)<br>Generate code into subfolder"]
    QA["3. QA STAGE (Parallel Audit + Tests)<br>Auto-heal up to 3 times"]
    HITL["4. HUMAN-IN-THE-LOOP<br>Manual terminal approval"]
    DEVOPS["5. DEVOPS & MEMORY COMPACTION<br>Compress blueprint -> Linear MCP Update -> Export telemetry.json"]

    BA --> DEV
    DEV --> QA
    QA -- "Delegation Protocol (Fail)" --> BA
    QA -- "Pass" --> HITL
    HITL -- "Approve" --> DEVOPS
```

### Stage Breakdown

1. **BA STAGE (Gemini)**: The pipeline starts by taking a raw human requirement. By leveraging the **Model Context Protocol (MCP)**, the BA agent can even read external documents like a Notion PRD directly. It then writes a highly technical checklist called `definitions_of_done.md` (DoD).
2. **DEV STAGE (agy)**: The Developer agent reads the DoD and writes the actual Go code into the `workspace/` folder.
3. **QA STAGE (Parallel Audit + Tests)**: The system automatically runs unit tests and strict security audits concurrently via goroutines. If the AI's code fails to compile, fails the tests, or contains malicious behavior, it triggers a **Self-Healing Loop** where the AI is fed the error logs and asked to try again. If it fails the maximum number of retries, it activates the **Delegation Protocol** to have the BA agent rewrite the requirements.
4. **HUMAN-IN-THE-LOOP (HITL)**: An engineer (you!) is prompted in the terminal. You review the code and type `y` to approve it for integration.
5. **DEVOPS & MEMORY COMPACTION**: The system generates release notes, exports `telemetry.json`, and leverages **MCP tools** to automatically update ticket trackers (like Linear) so you don't have to manually update project boards.
