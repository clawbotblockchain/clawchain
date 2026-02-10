#!/bin/bash
set -euo pipefail

# ClawChain Local Testnet Stop

HOME_BASE="${HOME}/.clawchain-localnet"
NUM_VALIDATORS=3

echo "=== Stopping ClawChain Local Testnet ==="

for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    PID_FILE="$HOME_BASE/node$i.pid"
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
            echo "Stopping node $i (PID: $PID)..."
            kill "$PID"
        else
            echo "Node $i already stopped"
        fi
        rm -f "$PID_FILE"
    else
        echo "No PID file for node $i"
    fi
done

echo "All nodes stopped."
