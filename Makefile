# ─────────────────────────────────────────────────────────────────────────────
# Harness Engineering — Makefile
# ─────────────────────────────────────────────────────────────────────────────
# Usage:
#   make update-docs          Regenerate README + documentation from source
#   make release              Build binary, tag version, commit & push
#   make build                Build the binary only
#   make help                 Show this help message

# ── Config ───────────────────────────────────────────────────────────────────
BINARY_NAME   := harness_bin
MAIN_PACKAGE  := .
VERSION_FILE  := .version

# Derive the next version. Reads from .version file, or falls back to latest git tag.
CURRENT_VERSION := $(shell cat $(VERSION_FILE) 2>/dev/null || git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
# Bump the patch segment automatically (e.g. v1.2.3 -> v1.2.4)
VERSION_PARTS   := $(subst ., ,$(subst v,,$(CURRENT_VERSION)))
MAJOR           := $(word 1,$(VERSION_PARTS))
MINOR           := $(word 2,$(VERSION_PARTS))
PATCH           := $(word 3,$(VERSION_PARTS))
NEXT_PATCH      := $(shell echo $$(($(PATCH)+1)))
NEXT_VERSION    := v$(MAJOR).$(MINOR).$(NEXT_PATCH)

# Build metadata
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# ── Targets ──────────────────────────────────────────────────────────────────

.PHONY: help build update-docs release docker-build docker-up docker-run docker-down

## help: print this help message
help:
	@echo ""
	@echo "  Harness Engineering — available make targets"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
	@echo ""

# ─────────────────────────────────────────────────────────────────────────────
## build: compile main.go into the harness_bin binary
# ─────────────────────────────────────────────────────────────────────────────
build:
	@echo "🔨 Building $(BINARY_NAME) (commit: $(GIT_COMMIT))..."
	@go build -ldflags="-X main.BuildVersion=$(NEXT_VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)" \
		-o $(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "✅ Build successful → ./$(BINARY_NAME)"

# ─────────────────────────────────────────────────────────────────────────────
## update-docs: scan main.go for changes and regenerate README + docs via AI
# ─────────────────────────────────────────────────────────────────────────────
update-docs:
	@echo "📚 Regenerating documentation from source code..."

	@# 1. Collect key source context
	@SOURCE_SUMMARY=$$(cat main.go harness_config.json go.mod internal/*/*.go); \
	README_CONTENT=$$(cat README.md); \
	\
	PROMPT="You are a senior technical writer. Below is the current state of the Harness Engineering Go orchestrator source code and its existing README.\
\n\nYour task:\
\n1. Update the README.md to accurately reflect the CURRENT architecture, pipeline stages, CLI flags, and config options found in main.go and internal packages.\
\n2. Keep the Mermaid pipeline diagram in sync with the Stage constants.\
\n3. Preserve the existing tone and structure — do not add marketing fluff.\
\n4. Output ONLY the final markdown content for README.md.\
\n\n=== main.go + internal/**/*.go + go.mod + harness_config.json ===\
\n$$SOURCE_SUMMARY\
\n\n=== CURRENT README.md ===\
\n$$README_CONTENT"; \
	\
	echo "🤖 Invoking BA agent to rewrite README.md..."; \
	gemini run "$$PROMPT" > README.md.tmp && mv README.md.tmp README.md && echo "✅ README.md updated." || echo "⚠️  BA agent unavailable — README.md unchanged."

	@# 2. Update English ebook chapters that describe config / stages
	@echo "📖 Updating English ebook docs (architecture & setup chapters)..."
	@for chapter in documentation/ebook/en/02_pipeline_architecture.md documentation/ebook/en/06_environment_setup.md; do \
		if [ ! -f "$$chapter" ]; then continue; fi; \
		SOURCE=$$(cat main.go harness_config.json internal/*/*.go); \
		CURRENT=$$(cat $$chapter); \
		PROMPT="You are a technical writer. Update the following ebook chapter to match the CURRENT source code. Do not change the tone or structure. Output ONLY the updated markdown.\n\n=== SOURCE CODE ===\n$$SOURCE\n\n=== CURRENT CHAPTER ===\n$$CURRENT"; \
		gemini run "$$PROMPT" > $$chapter.tmp && mv $$chapter.tmp $$chapter && echo "  ✅ $$chapter updated." || echo "  ⚠️  Skipped $$chapter (agent unavailable)."; \
	done

	@echo "\n🎯 Documentation update complete."

# ─────────────────────────────────────────────────────────────────────────────
## release: build binary, bump version, tag, commit, and push
# ─────────────────────────────────────────────────────────────────────────────
release: build
	@echo ""
	@echo "🚀 Preparing release $(NEXT_VERSION) (was $(CURRENT_VERSION))..."

	@# 1. Confirm there are no uncommitted changes (other than what we'll commit)
	@if [ -n "$$(git status --porcelain -- . ':!$(BINARY_NAME)' ':!.version' ':!README.md')" ]; then \
		echo "⚠️  Warning: uncommitted changes detected outside release artifacts. Proceeding anyway..."; \
	fi

	@# 2. Persist the new version
	@echo "$(NEXT_VERSION)" > $(VERSION_FILE)
	@echo "📌 Version bumped to $(NEXT_VERSION)"

	@# 3. Stage release artifacts
	@git add $(BINARY_NAME) $(VERSION_FILE) README.md
	@git add documentation/ || true

	@# 4. Commit
	@git commit -m "release: $(NEXT_VERSION) — built $(BUILD_TIME) [$(GIT_COMMIT)]" \
		--allow-empty
	@echo "✅ Release commit created."

	@# 5. Create annotated tag
	@git tag -a "$(NEXT_VERSION)" -m "Harness Engineering $(NEXT_VERSION)"
	@echo "🏷️  Tagged $(NEXT_VERSION)"

	@# 6. Push commit + tag
	@git push && git push origin "$(NEXT_VERSION)"
	@echo ""
	@echo "🎉 Release $(NEXT_VERSION) published successfully!"

# ─────────────────────────────────────────────────────────────────────────────
# Docker targets
# ─────────────────────────────────────────────────────────────────────────────

## docker-build: build Docker images for harness + ollama sidecar
docker-build:
	@echo "🐳 Building Docker images..."
	@docker compose build
	@echo "✅ Docker build complete."

## docker-up: start the full stack (ollama + harness) in detached mode
docker-up:
	@echo "🐳 Starting Ollama + Harness stack..."
	@docker compose up -d
	@echo "✅ Stack is running. Use 'docker compose logs -f' to follow output."

## docker-run: run a single task via Docker (usage: make docker-run TASK="your requirement")
docker-run:
	@if [ -z "$(TASK)" ]; then echo "❌ Usage: make docker-run TASK=\"your requirement\""; exit 1; fi
	@echo "🐳 Running task in Docker: $(TASK)"
	@docker compose run --rm harness --task "$(TASK)"

## docker-down: stop and remove all containers
docker-down:
	@echo "🐳 Stopping Docker stack..."
	@docker compose down
	@echo "✅ Stack stopped."
