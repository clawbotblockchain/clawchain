#!/bin/bash
set -euo pipefail

# ClawChain Validator Onboarding Script
# Download and run: curl -sSL https://clawchain.vsa.co.za/install.sh | bash

CHAIN_ID="clawchain-1"
BINARY="clawchaind"
BINARY_URL="https://github.com/clawbotblockchain/clawchain/releases/latest/download/clawchaind-linux-amd64"
GENESIS_URL="https://raw.githubusercontent.com/clawbotblockchain/clawchain-networks/main/clawchain-1/genesis.json"
PEERS_URL="https://raw.githubusercontent.com/clawbotblockchain/clawchain-networks/main/clawchain-1/peers.txt"
INSTALL_DIR="/usr/local/bin"
DATA_DIR="$HOME/.clawchain"
MIN_STAKE="100000"  # 100K CLAW

echo "============================================"
echo "   ClawChain Validator Onboarding"
echo "   Proof of Participation Blockchain"
echo "============================================"
echo ""

# Check prerequisites
echo "[1/7] Checking prerequisites..."
for cmd in curl jq; do
    if ! command -v "$cmd" &> /dev/null; then
        echo "Error: $cmd is required. Install with: sudo apt-get install $cmd"
        exit 1
    fi
done

# Check Go
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Installing Go 1.24..."
    wget -q "https://go.dev/dl/go1.24.12.linux-amd64.tar.gz" -O /tmp/go.tar.gz
    sudo tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.profile
fi
echo "  Go: $(go version)"

# Download binary
echo ""
echo "[2/7] Downloading ClawChain binary..."
if [ -f "$INSTALL_DIR/$BINARY" ]; then
    echo "  Binary already exists, checking version..."
    CURRENT=$($BINARY version 2>/dev/null || echo "unknown")
    echo "  Current: $CURRENT"
fi

curl -sSL "$BINARY_URL" -o "/tmp/$BINARY"
chmod +x "/tmp/$BINARY"
sudo mv "/tmp/$BINARY" "$INSTALL_DIR/$BINARY"
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
echo "[6/7] Configuring peers..."
PEERS=$(curl -sSL "$PEERS_URL" 2>/dev/null || echo "")
if [ -n "$PEERS" ]; then
    sed -i "s/persistent_peers = \"\"/persistent_peers = \"$PEERS\"/" "$DATA_DIR/config/config.toml"
    echo "  Peers configured"
else
    echo "  Warning: Could not fetch peers. Configure manually in $DATA_DIR/config/config.toml"
fi

# Configure node
sed -i 's/minimum-gas-prices = ""/minimum-gas-prices = "0aclaw"/' "$DATA_DIR/config/app.toml"
sed -i 's/timeout_commit = ".*"/timeout_commit = "3s"/' "$DATA_DIR/config/config.toml"

# Install systemd service
echo ""
echo "[7/7] Installing systemd service..."
sudo tee /etc/systemd/system/clawchaind.service > /dev/null <<EOF
[Unit]
Description=ClawChain Blockchain Node
After=network-online.target
Wants=network-online.target

[Service]
User=$USER
ExecStart=$INSTALL_DIR/$BINARY start --home $DATA_DIR
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
echo "  1. Request founder tokens at https://clawchain.vsa.co.za/request"
echo "     (Minimum $MIN_STAKE CLAW needed to stake)"
echo ""
echo "  2. Start syncing:"
echo "     sudo systemctl start clawchaind"
echo ""
echo "  3. Check sync status:"
echo "     curl -s http://localhost:26657/status | jq '.result.sync_info.catching_up'"
echo ""
echo "  4. Once synced and funded, create your validator:"
echo "     $BINARY tx staking create-validator \\"
echo "       --amount=${MIN_STAKE}000000000000000000aclaw \\"
echo "       --pubkey=\$($BINARY tendermint show-validator --home $DATA_DIR) \\"
echo "       --moniker=\"$MONIKER\" \\"
echo "       --chain-id=$CHAIN_ID \\"
echo "       --commission-rate=0.10 \\"
echo "       --commission-max-rate=0.20 \\"
echo "       --commission-max-change-rate=0.01 \\"
echo "       --min-self-delegation=1 \\"
echo "       --from=validator \\"
echo "       --keyring-backend=file \\"
echo "       --home=$DATA_DIR"
echo ""
echo "  5. Claim participation rewards:"
echo "     $BINARY tx participation claim-rewards \\"
echo "       --from=validator --keyring-backend=file --home=$DATA_DIR"
echo ""
