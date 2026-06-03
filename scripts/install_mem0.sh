#!/bin/bash

# Exit on any error
set -e

echo "📦 Initializing Mem0 local server..."

# Check if mem0-server directory exists
if [ ! -d "mem0-server" ]; then
    echo "⬇️ Adding mem0 as a git submodule..."
    git submodule add https://github.com/mem0ai/mem0.git mem0-server
else
    echo "✅ mem0-server already exists."
fi

# Initialize and update submodules
echo "🔄 Updating submodules..."
git submodule update --init --recursive

echo "⚙️ Setting up environment for Mem0..."
cd mem0-server/server
# If there is no .env file, try copying the example one, or just touch it
if [ ! -f ".env" ]; then
    if [ -f ".env.example" ]; then
        cp .env.example .env
        echo "📝 Copied .env.example to .env. Please configure it if needed."
    else
        touch .env
        echo "📝 Created empty .env file."
    fi
fi

echo "🚀 Mem0 installation complete! You can start it via 'make memory-up'"
