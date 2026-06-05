#!/usr/bin/env bash
set -e

echo "📦 Installing claude-tap from local submodule..."
if command -v uv &> /dev/null; then
    uv tool install ./claude-tap
else
    pip install -e ./claude-tap
fi
echo "✅ claude-tap successfully installed!"
