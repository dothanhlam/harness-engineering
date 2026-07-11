# Integrating Harness into a New Project (Submodule) & Organizing Epics

This guide shows how to embed the Harness Orchestration Engine into an existing
or new project as a git **submodule**, configure it correctly, and prepare/organize
**epics** for autonomous decomposition and implementation.

It is grounded in the actual CLI behavior (`main.go`), the epic orchestrator
(`internal/pipeline/epic.go`), and the config loader (`internal/config`).

---

## 1. How it actually works

- The CLI is **subcommand-based**: `harness init` and `harness run`.
  Go's `flag` package accepts both `-flag` and `--flag`, so either style works.
- `harness run -project-dir X` performs an **`os.Chdir(X)`** before doing anything
  else (`main.go`). Everything afterward is relative to *your* project: it reads
  `harness_config.json`, `.harness/rules.json`, and writes `memory/`, `workspace/`,
  and `telemetry.json` **inside your project** — not inside the framework.
- Config loads with graceful fallback (`config.LoadConfig`): if there is no
  `harness_config.json` in your project, it uses built-in defaults whose model
  paths are `~/ai_models/*.gguf` — **not** the `hf://…` paths shown in the README.

### ⚠️ Two gotchas
1. **`harness init` does NOT create a `harness_config.json`.** It only writes
   `.harness/rules.json` and `.agents/antigravity_dev_prompt.md`. A freshly
   `init`'d project therefore runs against the fallback `~/ai_models/*.gguf`
   models and will likely fail to find a model. **Copy `harness_config.json`
   into the project yourself** (step 4 below) or pass `-dev-model hf://…` flags
   on every run.
2. **Epics are Go-biased.** The decomposition prompt hard-codes "decompose into
   … Go sub-features" (`epic.go`). QA auto-detects Node/Python/Go, but the
   planner assumes Go modules. For non-Go projects, edit the `sysPrompt` in
   `epic.go`.

---

## 2. Integrate as a submodule

```bash
cd /path/to/my-project

# 1. Add the framework as a submodule (pin to a tag/commit for reproducibility)
git submodule add https://github.com/dothanhlam/harness-engineering.git .harness-framework
( cd .harness-framework && git checkout v0.0.5 )

# 2. Build the binary once
( cd .harness-framework && go build -o harness . )

# 3. Scaffold the project (creates .harness/rules.json + .agents/)
./.harness-framework/harness init -project-dir .

# 4. IMPORTANT: copy a config so it doesn't fall back to ~/ai_models defaults
cp .harness-framework/harness_config.json ./harness_config.json
```

Then edit `./harness_config.json` for your models. The `hf://…` paths are
downloaded and cached automatically under `~/.cache/huggingface/hub`.

### Single-task run

```bash
./.harness-framework/harness run -project-dir . \
  -task "Create a secure bcrypt hashing module"

# optional: pin the output folder instead of letting the BA derive it
#   -target workspace/security
```

### Updating the framework later

```bash
cd .harness-framework
git fetch && git checkout <newer-tag>
go build -o harness .
cd ..
git add .harness-framework   # commit the new submodule pointer
```

### What to commit vs ignore

Add to your project's `.gitignore`:

```
/workspace/      # generated code — keep or ignore per your workflow
/memory/         # generated DoD + blueprints
/telemetry.json
/.harness/       # keep if you customize rules.json; ignore otherwise
```

Commit: `harness_config.json`, `.agents/`, and the submodule pointer.

---

## 3. Prepare & organize epics

An **epic is a directory of requirement files**. `ExecuteBigEpic` reads **every
file** in that directory, concatenates them into one context, and asks the
DevOps/PM agent to emit a JSON plan of decoupled sub-tasks — each mapped to its
own `workspace/<name>` subfolder.

### Folder layout

```
my-project/
└── requirements/
    └── auth_epic/                 # <- pass this dir to -epic
        ├── 00_overview.md
        ├── 10_jwt_tokens.md
        ├── 20_password_reset.md
        └── 30_session_store.md
```

### Writing each requirement file

Plain markdown. For clean decomposition, make each file (or clearly-delimited
section) **one decoupled, single-responsibility feature**, and include a
**ticket ID** if you want it tracked (the planner extracts patterns like
`ENG-123` into `ticket_id`):

```markdown
# JWT Token Service  (ENG-142)

Implement a standalone Go package that issues and validates JWT access
tokens. No dependency on the password-reset or session modules.

## Requirements
- Issue HS256 tokens with configurable TTL
- Validate signature + expiry, return typed errors
- 100% unit-test coverage on issue/validate paths
```

Guidelines that map directly to planner behavior:

- **One feature per file/section** → one clean `workspace/<snake_case_name>`
  folder. Coupled requirements merge unpredictably.
- **Name the feature in the heading** → drives `task_name` + subfolder.
- **State "no dependency on X" explicitly** — parallel mode runs modules
  concurrently in isolated `memory/<task>/` dirs and assumes independence.
- Keep it Go-oriented (or edit the `sysPrompt` in `epic.go`).

### Run the epic

```bash
# Sequential — each sub-task runs the full BA→Dev→QA→HITL loop, one at a time
./.harness-framework/harness run -project-dir . -epic ./requirements/auth_epic/

# Parallel — Dev+QA fan out across modules, with a single HITL gate for all
./.harness-framework/harness run -project-dir . -epic ./requirements/auth_epic/ -parallel-epic
```

Choose **sequential** when modules build on each other or you want per-module
approval; choose **parallel** when modules are truly independent and you want
speed (it isolates memory per task to avoid `definitions_of_done.md` races).

---

## 4. Quick reference — `harness run` flags

| Flag | Purpose |
|---|---|
| `-project-dir` | Target project directory (the CLI `chdir`s into it) |
| `-task` | Single raw requirement → triggers the BA phase |
| `-target` | Explicit output subfolder (else derived from BA output) |
| `-epic` | Directory of requirement files → triggers the Epic Orchestrator |
| `-parallel-epic` | Run epic sub-tasks concurrently with isolated memory |
| `-force-regression` | Force QA regression tests |
| `-ba-agent` / `-ba-model` | Override BA agent / model |
| `-dev-agent` / `-dev-model` | Override Dev agent / model |
| `-devops-agent` / `-devops-model` | Override DevOps agent / model |
