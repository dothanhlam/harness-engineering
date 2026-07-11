# Chapter 3: Safety First - Guardrails & Self-Healing

When giving autonomous agents the ability to write code, safety and reliability are our highest priorities. We have built two primary defense mechanisms into the Harness.

## 1. The Self-Healing and Delegation Loops

AI makes mistakes. Syntax errors, failing tests, and misunderstood requirements are common. Our pipeline handles this gracefully:
* **The QA Self-Healing Loop**: If the test suite fails, the pipeline captures the exact compilation or test failure logs from `workspace/qa_error.log` and feeds them back to the Developer agent. The agent gets up to 3 attempts (`MaxRetries`) to fix its own code.
* **The Delegation Loop**: What if the developer agent fails 3 times? Instead of crashing, the Harness delegates the failure *back up the chain*. It activates the `BA_REFACTOR` stage, waking up the Business Analyst agent. The BA agent analyzes the failure logs and rewrites the `definitions_of_done.md` to clear up ambiguity, ensuring the developer has a better chance on the next cycle.

## 2. Governance & Security Audit Guardrails

Before any AI-generated code is even allowed to be compiled or tested, it must pass through the `AuditGeneratedCode` function in the `internal/qa` package. This runs as part of the `QA_TESTING` stage, in parallel with the test suite.

This function statically analyzes the AI's code for highly dangerous patterns. If any are found, the build fails immediately, and the AI is instructed to remove them during the self-healing loop.

**What do we scan for?**
* **Command Execution**: We block the `os/exec` package. The AI is not allowed to write code that executes arbitrary shell commands on our host machines.
* **Destructive Commands**: Strings like `rm -rf` are strictly forbidden.
* **Unauthorized File Manipulation**: The AI is blocked from using `os.Remove`, `os.RemoveAll`, or `os.Rename` to prevent it from accidentally (or maliciously) modifying system files outside its sandbox.
* **Hardcoded Credentials**: We scan for `password =`, `secret =`, and `aws_access_key` to ensure the AI doesn't hallucinate or leak sensitive authentication details into the source code.

**These rules are configurable.** When you run `harness init` on a project, the default rule set is written to `.harness/rules.json`. Edit that file to add project-specific bans or relax defaults — the pipeline loads it automatically at runtime. You can also exclude already-trusted modules from auditing via the `qa_ignore` list in `harness_config.json`.
