#!/usr/bin/env bash
set -euo pipefail

OLLAMA_HOST="${OLLAMA_HOST:-http://ollama:11434}"
OLLAMA_MODELS="${OLLAMA_MODELS:-hermes3:8b,gemma4:e4b}"

# ── Wait for Ollama to be ready ──────────────────────────────────────────────
echo "⏳ Waiting for Ollama at ${OLLAMA_HOST}..."
MAX_RETRIES=60
RETRY=0
until curl -sf "${OLLAMA_HOST}/api/tags" > /dev/null 2>&1; do
  RETRY=$((RETRY + 1))
  if [ "$RETRY" -ge "$MAX_RETRIES" ]; then
    echo "❌ Ollama did not become ready after ${MAX_RETRIES} attempts. Exiting."
    exit 1
  fi
  echo "   Attempt ${RETRY}/${MAX_RETRIES}..."
  sleep 2
done
echo "✅ Ollama is ready."

# ── Pull required models if not already present ─────────────────────────────
IFS=',' read -ra MODELS <<< "$OLLAMA_MODELS"
EXISTING_MODELS=$(curl -sf "${OLLAMA_HOST}/api/tags" | jq -r '.models[].name // empty' 2>/dev/null || echo "")

for MODEL in "${MODELS[@]}"; do
  MODEL=$(echo "$MODEL" | xargs)  # trim whitespace
  if echo "$EXISTING_MODELS" | grep -q "^${MODEL}"; then
    echo "✅ Model '${MODEL}' already available."
  else
    echo "📥 Pulling model '${MODEL}'... (this may take a while on first run)"
    curl -sf "${OLLAMA_HOST}/api/pull" \
      -d "{\"name\": \"${MODEL}\"}" \
      --no-buffer | while IFS= read -r line; do
        STATUS=$(echo "$line" | jq -r '.status // empty' 2>/dev/null)
        if [ -n "$STATUS" ]; then
          printf "\r   %s" "$STATUS"
        fi
      done
    echo ""
    echo "✅ Model '${MODEL}' pulled successfully."
  fi
done

echo ""
echo "🚀 Starting Harness Pipeline..."
exec ./harness_bin "$@"
