# ClawChain Validator Guide

This guide walks you through the complete process of setting up a ClawChain validator node from scratch.

**Chain ID:** `clawchain-1`
**Bond Denom:** `aclaw` (18 decimal places; 1 CLAW = 1,000,000,000,000,000,000 aclaw)
**Min Stake:** 100,000 CLAW (100000000000000000000000 aclaw)
**Max Validators:** 125
**Address Prefix:** `claw`

**Network Endpoints:**
- RPC: https://rpc.clawchain.vsa.co.za
- REST/LCD API: https://api.clawchain.vsa.co.za
- Explorer: https://clawchain.vsa.co.za
- GitHub: https://github.com/clawbotblockchain/clawchain

---

## Prerequisites

| Requirement | Minimum | Recommended |
|-------------|---------|-------------|
| CPU | 4 cores | 8+ cores |
| RAM | 8 GB | 16+ GB |
| Disk | 200 GB SSD | 500 GB NVMe |
| Bandwidth | 100 Mbps | 1 Gbps |
| OS | Ubuntu 22.04 LTS | Ubuntu 24.04 LTS |
| Go | 1.24+ | 1.24.12 |

You will also need: `git`, `curl`, `jq`, `make`, and `gcc`.

---

## Step 1: Server Setup

Update your system and install dependencies:

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y build-essential git curl jq lz4
```

### Install Go 1.24

```bash
GO_VERSION="1.24.12"
wget "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
rm "go${GO_VERSION}.linux-amd64.tar.gz"

# Add to your shell profile (~/.bashrc or ~/.profile)
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify
go version
# Expected: go version go1.24.12 linux/amd64
```

### Build clawchaind from source

```bash
cd $HOME
git clone https://github.com/clawbotblockchain/clawchain.git
cd clawchain
make install

# Verify
clawchaind version
```

The binary will be installed to `$HOME/go/bin/clawchaind`.

---

## Step 2: Initialize Your Node

Choose a moniker (validator name) that will identify your node on the network:

```bash
MONIKER="your-validator-name"
clawchaind init "$MONIKER" --chain-id clawchain-1
```

This creates the default configuration in `$HOME/.clawchain/`:
- `config/config.toml` -- Tendermint/CometBFT configuration
- `config/app.toml` -- Application configuration
- `config/genesis.json` -- Default genesis (will be replaced)
- `data/` -- Chain data directory

---

## Step 3: Get the Genesis File

Replace the default genesis file with the official ClawChain genesis:

**Option A: Download from the network repository**

```bash
curl -sSL https://raw.githubusercontent.com/clawbotblockchain/clawchain/main/config/genesis.json \
  -o $HOME/.clawchain/config/genesis.json
```

**Option B: Copy from an existing node**

If you have access to a running node, copy its genesis file:

```bash
scp user@existing-node:~/.clawchain/config/genesis.json $HOME/.clawchain/config/genesis.json
```

Verify the genesis file:

```bash
clawchaind genesis validate $HOME/.clawchain/config/genesis.json
```

---

## Step 4: Configure Persistent Peers

Edit `$HOME/.clawchain/config/config.toml` and set the `persistent_peers` field under the `[p2p]` section:

```toml
persistent_peers = "de57564ce9fb66a11ed626e842de9bbae3662dfe@seed0.clawchain.vsa.co.za:26656,50e9167c820a431a011d3f35b2835aa3239b46a6@seed1.clawchain.vsa.co.za:26756,38838db937b7e2cfe0d15bd16607d27e75b25bd8@seed2.clawchain.vsa.co.za:26856"
```

Or use `sed` to set it in one command:

```bash
PEERS="de57564ce9fb66a11ed626e842de9bbae3662dfe@seed0.clawchain.vsa.co.za:26656,50e9167c820a431a011d3f35b2835aa3239b46a6@seed1.clawchain.vsa.co.za:26756,38838db937b7e2cfe0d15bd16607d27e75b25bd8@seed2.clawchain.vsa.co.za:26856"
sed -i "s/^persistent_peers *=.*/persistent_peers = \"$PEERS\"/" $HOME/.clawchain/config/config.toml
```

### Additional recommended settings

Set minimum gas prices in `$HOME/.clawchain/config/app.toml`:

```bash
sed -i 's/^minimum-gas-prices *=.*/minimum-gas-prices = "0aclaw"/' $HOME/.clawchain/config/app.toml
```

Enable the REST API (useful for monitoring):

```bash
sed -i '/^\[api\]$/,/^\[/{s/enable = false/enable = true/}' $HOME/.clawchain/config/app.toml
```

Enable Prometheus metrics:

```bash
sed -i 's/^prometheus *=.*/prometheus = true/' $HOME/.clawchain/config/config.toml
```

---

## Step 5: Sync the Chain

Start the node to begin syncing:

```bash
clawchaind start
```

In a separate terminal, check sync status:

```bash
curl -s http://localhost:26657/status | jq '.result.sync_info'
```

When `catching_up` is `false`, your node is fully synced. This may take anywhere from a few minutes to several hours depending on chain height and your connection speed.

**Do not proceed to create your validator until the node is fully synced.**

---

## Step 6: Create Your Validator Key

Create a new key pair that will control your validator:

```bash
clawchaind keys add validator --keyring-backend file
```

**IMPORTANT:** Save the mnemonic phrase in a secure, offline location. This is the only way to recover your validator key. If you lose it, you lose control of your validator and staked funds.

To import an existing mnemonic instead:

```bash
clawchaind keys add validator --recover --keyring-backend file
```

View your validator address:

```bash
clawchaind keys show validator -a --keyring-backend file
# Output: claw1...
```

---

## Step 7: Get Funded

To create a validator, you need a minimum of **100,000 CLAW** (100000000000000000000000 aclaw).

### Application process:

1. Open a [Validator Application](https://github.com/clawbotblockchain/clawchain/issues/new?template=validator-application.yml) issue on GitHub.
2. Include your `claw1...` address from Step 6.
3. The team will review your application.
4. Once approved, the founder account will send you 100,000 CLAW.

Check your balance:

```bash
clawchaind query bank balances $(clawchaind keys show validator -a --keyring-backend file)
```

Or via the REST API:

```bash
curl -s "https://api.clawchain.vsa.co.za/cosmos/bank/v1beta1/balances/YOUR_ADDRESS"
```

---

## Step 8: Submit Create-Validator Transaction

Once your node is synced and your account is funded, create your validator:

```bash
clawchaind tx staking create-validator \
  --amount=100000000000000000000000aclaw \
  --pubkey=$(clawchaind tendermint show-validator) \
  --moniker="your-validator-name" \
  --chain-id=clawchain-1 \
  --commission-rate=0.10 \
  --commission-max-rate=0.20 \
  --commission-max-change-rate=0.01 \
  --min-self-delegation=1 \
  --from=validator \
  --keyring-backend=file \
  --fees=0aclaw \
  --yes
```

**Parameter explanations:**

| Parameter | Value | Description |
|-----------|-------|-------------|
| `--amount` | 100000000000000000000000aclaw | 100,000 CLAW to stake (18 decimal places) |
| `--commission-rate` | 0.10 | 10% commission on delegator rewards |
| `--commission-max-rate` | 0.20 | Maximum commission you can ever set (20%) |
| `--commission-max-change-rate` | 0.01 | Max daily commission change (1%) |
| `--min-self-delegation` | 1 | Minimum self-delegation in CLAW |

Verify your validator is in the active set:

```bash
clawchaind query staking validators --output json | jq '.validators[] | select(.description.moniker=="your-validator-name")'
```

Check on the explorer: https://clawchain.vsa.co.za

---

## Step 9: Set Up systemd Service

Running your validator as a systemd service ensures it restarts automatically on crashes and server reboots.

Create the service file:

```bash
sudo tee /etc/systemd/system/clawchaind.service > /dev/null <<EOF
[Unit]
Description=ClawChain Blockchain Node
After=network-online.target
Wants=network-online.target

[Service]
User=$USER
ExecStart=$(which clawchaind) start --home $HOME/.clawchain
Restart=always
RestartSec=3
LimitNOFILE=65535

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=clawchaind

[Install]
WantedBy=multi-user.target
EOF
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable clawchaind
sudo systemctl start clawchaind
```

Check the service status:

```bash
sudo systemctl status clawchaind
```

View logs:

```bash
journalctl -u clawchaind -f
```

---

## Step 10: Monitoring and Maintenance

### Check validator status

```bash
# Is your validator signing blocks?
clawchaind query slashing signing-info $(clawchaind tendermint show-validator)

# Check your validator's voting power
curl -s http://localhost:26657/status | jq '.result.validator_info'
```

### Claim participation rewards

ClawChain uses a Proof of Participation model with daily rewards split between validators (40%) and workers (60%). Claim your validator rewards periodically:

```bash
clawchaind tx participation claim-rewards \
  --from=validator \
  --keyring-backend=file \
  --chain-id=clawchain-1 \
  --fees=0aclaw \
  --yes
```

### Unjail your validator

If your validator gets jailed for downtime, unjail it after resolving the issue:

```bash
clawchaind tx slashing unjail \
  --from=validator \
  --keyring-backend=file \
  --chain-id=clawchain-1 \
  --fees=0aclaw \
  --yes
```

### Update your validator description

```bash
clawchaind tx staking edit-validator \
  --moniker="new-name" \
  --details="Description of your validator" \
  --website="https://your-website.com" \
  --identity="YOUR_KEYBASE_ID" \
  --from=validator \
  --keyring-backend=file \
  --chain-id=clawchain-1 \
  --fees=0aclaw \
  --yes
```

### Node upgrades

When a new version of clawchaind is released:

```bash
# Stop the service
sudo systemctl stop clawchaind

# Pull latest code and rebuild
cd $HOME/clawchain
git pull
make install

# Restart
sudo systemctl start clawchaind

# Verify
clawchaind version
journalctl -u clawchaind -f
```

### Backup

Always maintain backups of these critical files:

- `$HOME/.clawchain/config/priv_validator_key.json` -- Your validator signing key
- `$HOME/.clawchain/config/node_key.json` -- Your node identity key
- Your key mnemonic (stored offline)

**Never share your `priv_validator_key.json` or run it on two nodes simultaneously.** Running the same validator key on multiple nodes will result in double-signing and permanent slashing.

---

## Useful Commands Reference

```bash
# Check sync status
curl -s http://localhost:26657/status | jq '.result.sync_info.catching_up'

# View all validators
clawchaind query staking validators

# View your validator info
clawchaind query staking validator $(clawchaind keys show validator --bech val -a --keyring-backend file)

# Check account balance
clawchaind query bank balances $(clawchaind keys show validator -a --keyring-backend file)

# Delegate more tokens
clawchaind tx staking delegate $(clawchaind keys show validator --bech val -a --keyring-backend file) \
  AMOUNT_IN_ACLAW \
  --from=validator --keyring-backend=file --chain-id=clawchain-1 --fees=0aclaw --yes

# View governance proposals
clawchaind query gov proposals

# Vote on a proposal
clawchaind tx gov vote PROPOSAL_ID yes \
  --from=validator --keyring-backend=file --chain-id=clawchain-1 --fees=0aclaw --yes

# View participation module params
clawchaind query participation params
```

---

## Troubleshooting

### Node fails to start

- Check logs: `journalctl -u clawchaind -n 100 --no-pager`
- Verify genesis file: `clawchaind genesis validate`
- Ensure ports 26656 (P2P) and 26657 (RPC) are open in your firewall

### Node is not syncing

- Verify persistent_peers are configured correctly in `config.toml`
- Check that port 26656 is reachable from the internet
- Try restarting: `sudo systemctl restart clawchaind`

### Validator is jailed

- Check the reason: `clawchaind query slashing signing-info $(clawchaind tendermint show-validator)`
- Common causes: extended downtime, missed blocks
- Fix the underlying issue, then unjail (see Step 10)

### "account not found" error when creating validator

- Your account has no balance. Ensure your validator application is approved and tokens have been sent.
- Check your address: `clawchaind keys show validator -a --keyring-backend file`
- Verify on explorer: https://clawchain.vsa.co.za

---

## Security Best Practices

1. **Use a firewall.** Only expose ports 26656 (P2P) and optionally 26657 (RPC), 1317 (REST API), 9090 (gRPC).
2. **Run as a non-root user.** Create a dedicated `clawchain` user for the service.
3. **Enable automatic security updates.** Use `unattended-upgrades` on Ubuntu.
4. **Set up monitoring and alerting.** Use Prometheus, Grafana, and alerting for downtime.
5. **Use a sentry node architecture** for production validators to protect against DDoS attacks.
6. **Keep your validator key secure.** Consider using a hardware security module (HSM) or remote signer like Horcrux/TMKMS for mainnet.
7. **Regularly update** to the latest clawchaind version.

---

## Getting Help

- GitHub Issues: https://github.com/clawbotblockchain/clawchain/issues
- Explorer: https://clawchain.vsa.co.za
