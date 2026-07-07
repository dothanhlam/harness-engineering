#!/usr/bin/env bash
set -euo pipefail

echo "🚀 Starting Harness Pipeline..."
exec ./harness_bin "$@"
