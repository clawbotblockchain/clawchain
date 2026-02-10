#!/bin/bash
set -euo pipefail

# ClawChain Worker Setup Script
# Sets up a new worker node for heartbeat-based participation rewards.
# No tokens, no gas fees, no minimum stake required.

CHAIN_ID="clawchain-1"
RPC_URL="https://rpc.clawchain.vsa.co.za:443"
BINARY="clawchaind"
HEARTBEAT_INTERVAL=300  # 5 minutes

echo "=========================================="
echo "   ClawChain Worker Setup"
echo "   Chain ID: $CHAIN_ID"
echo "=========================================="
echo ""

# Check if clawchaind binary is available
if ! command -v "$BINARY" &>/dev/null; then
    echo "ERROR: '$BINARY' not found in PATH."
    echo ""
    echo "Build it from source:"
    echo "  git clone https://github.com/clawbotblockchain/clawchain.git"
    echo "  cd clawchain"
    echo "  go build -o clawchaind ./cmd/clawchaind"
    echo "  sudo mv clawchaind /usr/local/bin/"
    echo ""
    echo "Requires Go 1.24+"
    exit 1
fi

echo "Found: $(command -v $BINARY)"
echo ""

# Get worker name
WORKER_NAME="${1:-}"
if [ -z "$WORKER_NAME" ]; then
    read -rp "Enter a name for your worker (e.g. my-bot): " WORKER_NAME
fi

if [ -z "$WORKER_NAME" ]; then
    echo "ERROR: Worker name is required."
    exit 1
fi

echo ""
echo "--- Step 1: Create key ---"
echo ""

# Check if key already exists
if $BINARY keys show "$WORKER_NAME" --keyring-backend test &>/dev/null; then
    echo "Key '$WORKER_NAME' already exists."
    WORKER_ADDR=$($BINARY keys show "$WORKER_NAME" -a --keyring-backend test)
else
    echo "Creating key '$WORKER_NAME'..."
    $BINARY keys add "$WORKER_NAME" --keyring-backend test
    WORKER_ADDR=$($BINARY keys show "$WORKER_NAME" -a --keyring-backend test)
fi

echo ""
echo "Worker address: $WORKER_ADDR"

echo ""
echo "--- Step 2: Configure node ---"
echo ""

$BINARY config set client node "$RPC_URL"
echo "RPC endpoint: $RPC_URL"

$BINARY config set client chain-id "$CHAIN_ID"
echo "Chain ID: $CHAIN_ID"

echo ""
echo "--- Step 3: Register as worker ---"
echo ""
echo "Registering '$WORKER_NAME' on ClawChain..."

$BINARY tx participation register-worker \
    --name "$WORKER_NAME" \
    --from "$WORKER_NAME" \
    --chain-id "$CHAIN_ID" \
    --keyring-backend test \
    --yes

echo ""
echo "--- Step 4: Start sending heartbeats ---"
echo ""
echo "You can run heartbeats manually:"
echo ""
echo "  while true; do"
echo "    $BINARY tx participation heartbeat --from $WORKER_NAME --chain-id $CHAIN_ID --keyring-backend test --yes"
echo "    sleep $HEARTBEAT_INTERVAL"
echo "  done"
echo ""
echo "Or set up a systemd service (recommended):"
echo ""
echo "Create this file at /etc/systemd/system/clawchain-worker.service"
echo "(requires sudo — run these commands yourself):"
echo ""
cat <<UNIT
[Unit]
Description=ClawChain Worker Heartbeat ($WORKER_NAME)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$USER
ExecStart=/bin/bash -c 'while true; do $BINARY tx participation heartbeat --from $WORKER_NAME --chain-id $CHAIN_ID --keyring-backend test --yes 2>&1 | tail -1; sleep $HEARTBEAT_INTERVAL; done'
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
UNIT

echo ""
echo "Then enable and start it:"
echo "  sudo systemctl daemon-reload"
echo "  sudo systemctl enable clawchain-worker"
echo "  sudo systemctl start clawchain-worker"
echo ""
echo "--- Check your rewards ---"
echo ""
echo "  $BINARY q participation worker $WORKER_ADDR --node $RPC_URL"
echo "  $BINARY q participation worker-rewards $WORKER_ADDR --node $RPC_URL"
echo ""
echo "--- Claim rewards ---"
echo ""
echo "  $BINARY tx participation claim-worker-rewards --from $WORKER_NAME --chain-id $CHAIN_ID --keyring-backend test --yes"
echo ""
echo "=========================================="
echo "   Worker setup complete!"
echo "   Earning: proportional share of 22.5M CLAW/day"
echo "   More workers = smaller individual share"
echo "   More heartbeats = bigger share"
echo "=========================================="
