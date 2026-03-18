#!/bin/bash
set -euo pipefail

# ClawChain Validator Onboarding Script
# Usage: bash become-validator.sh

CHAIN_ID="clawchain-1"
BINARY="clawchaind"
REPO_URL="https://github.com/clawbotblockchain/clawchain"
GENESIS_URL="https://raw.githubusercontent.com/clawbotblockchain/clawchain/main/networks/clawchain-1/genesis.json"
PEERS_URL="https://raw.githubusercontent.com/clawbotblockchain/clawchain/main/networks/clawchain-1/peers.txt"
SEED_PEER="de57564ce9fb66a11ed626e842de9bbae3662dfe@seed.clawchain.vsa.co.za:26656"
DATA_DIR="$HOME/.clawchain"
MIN_STAKE="100000"  # 100K CLAW

echo "============================================"
echo "   ClawChain Validator Onboarding"
echo "   Proof of Participation Blockchain"
echo "============================================"
echo ""

# Check prerequisites
echo "[1/7] Checking prerequisites..."
for cmd in curl jq git make; do
    if ! command -v "$cmd" &> /dev/null; then
        echo "Error: $cmd is required. Install with: sudo apt-get install -y $cmd"
        exit 1
    fi
done

# Check Go
if ! command -v go &> /dev/null; then
    echo ""
    echo "Go is not installed. Install Go 1.24+ before continuing:"
    echo "  wget https://go.dev/dl/go1.24.12.linux-amd64.tar.gz"
    echo "  sudo tar -C /usr/local -xzf go1.24.12.linux-amd64.tar.gz"
    echo "  echo 'export PATH=\$PATH:/usr/local/go/bin:\$HOME/go/bin' >> ~/.profile"
    echo "  source ~/.profile"
    exit 1
fi
echo "  Go: $(go version)"

# Build from source
echo ""
echo "[2/7] Building ClawChain from source..."
BUILD_DIR="/tmp/clawchain-build"
if [ -d "$BUILD_DIR" ]; then
    echo "  Updating existing source..."
    cd "$BUILD_DIR" && git pull --quiet
else
    echo "  Cloning repository..."
    git clone --quiet "$REPO_URL" "$BUILD_DIR"
    cd "$BUILD_DIR"
fi

echo "  Building clawchaind..."
go build -o "$HOME/go/bin/$BINARY" ./cmd/clawchaind
echo "  Installed: $($BINARY version 2>/dev/null || echo 'ok')"

# Initialize node
echo ""
echo "[3/7] Initializing node..."
read -p "Enter your validator name (moniker): " MONIKER
$BINARY init "$MONIKER" --chain-id "$CHAIN_ID" --home "$DATA_DIR" 2>/dev/null || true

# Create or import key
echo ""
echo "[4/7] Setting up validator key..."
echo "Options:"
echo "  1. Create new key"
echo "  2. Import existing key (mnemonic)"
read -p "Choose (1/2): " KEY_CHOICE

if [ "$KEY_CHOICE" = "2" ]; then
    $BINARY keys add validator --recover --keyring-backend file --home "$DATA_DIR"
else
    $BINARY keys add validator --keyring-backend file --home "$DATA_DIR"
fi

VALIDATOR_ADDR=$($BINARY keys show validator -a --keyring-backend file --home "$DATA_DIR")
echo ""
echo "Your validator address: $VALIDATOR_ADDR"

# Download genesis
echo ""
echo "[5/7] Downloading genesis file..."
curl -sSL "$GENESIS_URL" -o "$DATA_DIR/config/genesis.json"
echo "  Genesis downloaded"

# Configure peers
echo ""
echo "[6/7] Configuring seed node..."
# Fetch latest peers from repo, fall back to hardcoded seed
PEERS=$(curl -sSL "$PEERS_URL" 2>/dev/null | grep -v '^#' | grep -v '^$' | tr '\n' ',' | sed 's/,$//' || echo "$SEED_PEER")
if [ -z "$PEERS" ]; then
    PEERS="$SEED_PEER"
fi
sed -i "s|persistent_peers = \".*\"|persistent_peers = \"$PEERS\"|" "$DATA_DIR/config/config.toml"
echo "  Peers: $PEERS"

# Configure node
sed -i 's/minimum-gas-prices = ""/minimum-gas-prices = "0aclaw"/' "$DATA_DIR/config/app.toml"
sed -i 's/timeout_commit = ".*"/timeout_commit = "3s"/' "$DATA_DIR/config/config.toml"
# Enable API for monitoring
sed -i 's/enable = false/enable = true/' "$DATA_DIR/config/app.toml"

# Install systemd service
echo ""
echo "[7/7] Installing systemd service..."
echo "  (requires sudo)"
sudo tee /etc/systemd/system/clawchaind.service > /dev/null <<EOF
[Unit]
Description=ClawChain Blockchain Node
After=network-online.target
Wants=network-online.target

[Service]
User=$USER
ExecStart=$(which $BINARY) start --home $DATA_DIR
Restart=always
RestartSec=3
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable clawchaind

echo ""
echo "============================================"
echo "   Setup Complete!"
echo "============================================"
echo ""
echo "Your validator address: $VALIDATOR_ADDR"
echo ""
echo "Next steps:"
echo ""
echo "  1. Apply for validator approval:"
echo "     Open an issue at $REPO_URL/issues/new"
echo "     Use the 'Validator Application' template"
echo "     Include your address: $VALIDATOR_ADDR"
echo ""
echo "  2. Start syncing:"
echo "     sudo systemctl start clawchaind"
echo ""
echo "  3. Check sync status:"
echo "     curl -s http://localhost:26657/status | jq '.result.sync_info.catching_up'"
echo "     (wait until catching_up = false)"
echo ""
echo "  4. Once approved, funded, and synced, create your validator:"
echo "     $BINARY tx staking create-validator \\"
echo "       --amount=${MIN_STAKE}000000000000000000000aclaw \\"
echo "       --pubkey=\$($BINARY tendermint show-validator --home $DATA_DIR) \\"
echo "       --moniker=\"$MONIKER\" \\"
echo "       --chain-id=$CHAIN_ID \\"
echo "       --commission-rate=0.10 \\"
echo "       --commission-max-rate=0.20 \\"
echo "       --commission-max-change-rate=0.01 \\"
echo "       --min-self-delegation=1 \\"
echo "       --from=validator \\"
echo "       --keyring-backend=file \\"
echo "       --home=$DATA_DIR \\"
echo "       --fees=0aclaw --yes"
echo ""
echo "  Resources:"
echo "     Explorer:  https://clawchain.vsa.co.za"
echo "     RPC:       https://rpc.clawchain.vsa.co.za"
echo "     API:       https://api.clawchain.vsa.co.za"
echo "     Guide:     $REPO_URL/blob/main/docs/VALIDATOR_GUIDE.md"
echo ""
