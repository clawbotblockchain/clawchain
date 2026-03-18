#!/bin/bash
# ClawChain Gateway startup script
set -euo pipefail

GATEWAY_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$GATEWAY_DIR"

# Load environment
set -a
source "$GATEWAY_DIR/.env"
set +a

# Ensure PATH includes Go binaries
export PATH="/home/claude/go/bin:/home/claude/.local/go/bin:$PATH"

# Run as a Python package from parent dir
cd "$(dirname "$GATEWAY_DIR")"
exec "$GATEWAY_DIR/.venv/bin/uvicorn" gateway.main:app \
    --host 127.0.0.1 \
    --port 8400 \
    --log-level info
