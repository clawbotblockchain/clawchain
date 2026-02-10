#!/bin/bash
set -euo pipefail

# ClawChain Validator Deployment Script
# Usage: bash scripts/deploy-validator.sh <server-ip> <validator-name> [genesis-file]

if [ $# -lt 2 ]; then
    echo "Usage: $0 <server-ip> <validator-name> [genesis-file]"
    echo ""
    echo "Arguments:"
    echo "  server-ip       IP or hostname of the target server"
    echo "  validator-name  Moniker for this validator"
    echo "  genesis-file    Optional path to genesis.json (default: fetch from seed)"
    exit 1
fi

SERVER_IP="$1"
VALIDATOR_NAME="$2"
GENESIS_FILE="${3:-}"

BINARY="clawchaind"
BINARY_PATH="build/${BINARY}-linux-amd64"
CHAIN_ID="clawchain-1"
REMOTE_HOME="/opt/clawchain/.clawchain"
REMOTE_USER="${REMOTE_USER:-root}"

echo "=== Deploying ClawChain Validator ==="
echo "Server: $SERVER_IP"
echo "Validator: $VALIDATOR_NAME"
echo ""

# Check binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo "Binary not found at $BINARY_PATH"
    echo "Run: make build-linux"
    exit 1
fi

# Copy binary to server
echo "Copying binary..."
scp "$BINARY_PATH" "$REMOTE_USER@$SERVER_IP:/usr/local/bin/$BINARY"
ssh "$REMOTE_USER@$SERVER_IP" "chmod +x /usr/local/bin/$BINARY"

# Initialize node
echo "Initializing node..."
ssh "$REMOTE_USER@$SERVER_IP" <<REMOTE
set -euo pipefail

# Init chain
$BINARY init "$VALIDATOR_NAME" --chain-id "$CHAIN_ID" --home "$REMOTE_HOME" 2>/dev/null || true

# Create validator key
$BINARY keys add validator --keyring-backend file --home "$REMOTE_HOME" 2>/dev/null || echo "Key already exists"
REMOTE

# Copy genesis file
if [ -n "$GENESIS_FILE" ]; then
    echo "Copying genesis file..."
    scp "$GENESIS_FILE" "$REMOTE_USER@$SERVER_IP:$REMOTE_HOME/config/genesis.json"
fi

# Copy systemd service
echo "Installing systemd service..."
scp config/systemd/clawchaind.service "$REMOTE_USER@$SERVER_IP:/etc/systemd/system/clawchaind.service"
ssh "$REMOTE_USER@$SERVER_IP" <<REMOTE
set -euo pipefail

# Configure node
CONFIG="$REMOTE_HOME/config/config.toml"
APP_TOML="$REMOTE_HOME/config/app.toml"

# Set minimum gas prices
sed -i 's/minimum-gas-prices = ""/minimum-gas-prices = "0aclaw"/' "\$APP_TOML"

# Enable API
sed -i '/^\[api\]$/,/^\[/{s/enable = false/enable = true/}' "\$APP_TOML"

# Enable gRPC
sed -i '/^\[grpc\]$/,/^\[/{s/enable = false/enable = true/}' "\$APP_TOML"

# Set block time
sed -i 's/timeout_commit = ".*"/timeout_commit = "3s"/' "\$CONFIG"

# Enable prometheus
sed -i 's/prometheus = false/prometheus = true/' "\$CONFIG"

# Reload and enable service
systemctl daemon-reload
systemctl enable clawchaind

echo ""
echo "Node initialized. Validator address:"
$BINARY keys show validator -a --keyring-backend file --home "$REMOTE_HOME" 2>/dev/null || echo "(run 'clawchaind keys show validator' manually)"
REMOTE

echo ""
echo "=== Deployment Complete ==="
echo ""
echo "Next steps on the server:"
echo "  1. Configure persistent_peers in $REMOTE_HOME/config/config.toml"
echo "  2. Copy the final genesis.json if not done"
echo "  3. Start: systemctl start clawchaind"
echo "  4. Check: journalctl -u clawchaind -f"
echo "  5. Create validator tx after chain starts"
