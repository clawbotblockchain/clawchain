#!/bin/bash
set -euo pipefail

# ClawChain Server Setup Script
# Run on each Ubuntu server to install prerequisites

echo "=== ClawChain Server Setup ==="
echo "Installing Go 1.24+ and build dependencies..."
echo ""

# Install build dependencies
sudo apt-get update
sudo apt-get install -y build-essential git curl wget jq make gcc

# Install Go 1.24.x
GO_VERSION="1.24.12"
GO_ARCH="amd64"
GO_OS="linux"
GO_TAR="go${GO_VERSION}.${GO_OS}-${GO_ARCH}.tar.gz"

if command -v go &> /dev/null; then
    CURRENT_GO=$(go version | awk '{print $3}' | sed 's/go//')
    echo "Go $CURRENT_GO already installed"
    # Check if version is sufficient
    MAJOR=$(echo "$CURRENT_GO" | cut -d. -f1)
    MINOR=$(echo "$CURRENT_GO" | cut -d. -f2)
    if [ "$MAJOR" -ge 1 ] && [ "$MINOR" -ge 24 ]; then
        echo "Go version is sufficient, skipping install"
    else
        echo "Upgrading Go..."
        sudo rm -rf /usr/local/go
        wget -q "https://go.dev/dl/${GO_TAR}" -O "/tmp/${GO_TAR}"
        sudo tar -C /usr/local -xzf "/tmp/${GO_TAR}"
        rm "/tmp/${GO_TAR}"
    fi
else
    echo "Installing Go ${GO_VERSION}..."
    wget -q "https://go.dev/dl/${GO_TAR}" -O "/tmp/${GO_TAR}"
    sudo tar -C /usr/local -xzf "/tmp/${GO_TAR}"
    rm "/tmp/${GO_TAR}"
fi

# Add Go to PATH
if ! grep -q '/usr/local/go/bin' ~/.profile 2>/dev/null; then
    echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.profile
fi
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin

echo "Go version: $(go version)"

# Create clawchain user (optional, for production)
if ! id -u clawchain &>/dev/null 2>&1; then
    echo "Creating clawchain system user..."
    sudo useradd -m -s /bin/bash clawchain || true
fi

# Create data directory
sudo mkdir -p /opt/clawchain
sudo chown "$(whoami)" /opt/clawchain

echo ""
echo "=== Server Setup Complete ==="
echo "Next steps:"
echo "  1. Build: make build-linux"
echo "  2. Deploy: bash scripts/deploy-validator.sh <server-ip> <validator-name>"
