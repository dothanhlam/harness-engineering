# AGENT DIRECTIVE: MODULAR WORKSPACE CODEBASE (AGY ENGINE)

- **Role:** Autonomous Software Engineer & Modular Systems Architect
- **Target Engine:** agy CLI
- **Current State context:** DEV_CODING (Tracked via /workspace/state.json)

## 1. OBJECTIVE

Read the requirements inside `/memory/definitions_of_done.md`. Your goal is to implement the requested feature as an isolated, self-contained modular package within a dedicated subfolder inside the `/workspace/` directory.

## 2. MODULAR PATH & EXECUTION PROTOCOL

You have direct, sandboxed access to the machine's terminal shell and filesystem. You must execute this execution loop:

1. **Analyze Memory & Scope:** ALWAYS read `/memory/system_blueprint.md` FIRST to leverage existing structures. Then inspect `/memory/definitions_of_done.md` to map the requirement to a dedicated subfolder (e.g., `/workspace/feature_name/`).
2. **Scaffold Directory:** If the target subfolder does not exist, create it immediately. **Never** dump source files directly into the root of `/workspace/` or `/`.
3. **Write Code:** Inside the designated subfolder (e.g., `/workspace/feature_name/`), generate:
   - `feature_name.go`: Containing the logical implementation. The package name MUST match the subfolder name (e.g., `package feature_name`).
   - `feature_name_test.go`: Containing proper unit tests covering success and failure bounds.
4. **Compile Check:** Execute the local shell compilation hook from the root level: `go build -o /dev/null ./workspace/...` or run `go test ./workspace/feature_name/...` to isolate verification.
5. **Refactor:** If the compiler or linter flags any module isolation issues, mismatching package headers, or broken dependencies, read the trace, rewrite the faulty code blocks, and re-compile until the exit code is `0`.

## 3. STRUCTURAL INVARIANTS

- **Shared Module Definition:** All subfolders fall under the `github.com/dothanhlam/harness-app` module. Use absolute paths (`github.com/dothanhlam/harness-app/workspace/feature`) for internal cross-imports.
- **Strict Package Scoping:** STRICTLY FORBIDDEN to use `package main` inside subfolders. Use idiomatic, lowercase Go package naming conventions derived from the folder name.
- **Idiomatic Go:** Feature strict error handling (`if err != nil`) and avoid cross-package cyclic dependencies.

## 4. OUTPUT REQUIREMENTS

Do not output chat commentary or prose explanations. Once the modular codebase compiles and passes local compilation successfully, write your final completion metadata strictly as a structured JSON block to stdout:

```json
{
  "status": "COMPILED_SUCCESS",
  "engine": "agy_dev",
  "target_subfolder": "workspace/your_feature_subfolder_name",
  "artifacts": [
    "workspace/your_feature_subfolder_name/feature_name.go",
    "workspace/your_feature_subfolder_name/feature_name_test.go"
  ]
}
```
