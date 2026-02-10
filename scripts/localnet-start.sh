#!/bin/bash
set -euo pipefail

# ClawChain Local Testnet Start
# Starts 3 validator nodes in the background

BINARY="clawchaind"
HOME_BASE="${HOME}/.clawchain-localnet"
NUM_VALIDATORS=3
LOG_DIR="$HOME_BASE/logs"

mkdir -p "$LOG_DIR"

echo "=== Starting ClawChain Local Testnet ==="

# Check that nodes are initialized
if [ ! -d "$HOME_BASE/node0" ]; then
    echo "Error: Testnet not initialized. Run: make localnet-init"
    exit 1
fi

# Start each validator
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    NODE_HOME="$HOME_BASE/node$i"
    LOG_FILE="$LOG_DIR/node$i.log"

    echo "Starting node $i..."
    $BINARY start --home "$NODE_HOME" > "$LOG_FILE" 2>&1 &
    echo $! > "$HOME_BASE/node$i.pid"
    echo "  PID: $(cat "$HOME_BASE/node$i.pid") | Log: $LOG_FILE"
done

echo ""
echo "=== All nodes started ==="
echo ""
echo "Endpoints:"
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    RPC_PORT=$((26657 + i * 100))
    API_PORT=$((1317 + i * 100))
    echo "  Node $i: RPC=http://127.0.0.1:$RPC_PORT API=http://127.0.0.1:$API_PORT"
done
echo ""
echo "Check status: curl http://127.0.0.1:26657/status | jq '.result.sync_info'"
echo "Stop: make localnet-stop"
echo "Logs: tail -f $LOG_DIR/node0.log"
