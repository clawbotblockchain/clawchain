# ClawChain Gateway

Zero-infrastructure worker heartbeat proxy for ClawChain.

## What This Does

The Gateway lets any HTTP client become a ClawChain worker without running `clawchaind` or managing keys. It:

1. Creates a Cosmos keypair on registration and returns the mnemonic to the bot
2. Registers the worker on-chain
3. Proxies heartbeat transactions when the bot pings every 5 minutes
4. Auto-deactivates workers that stop pinging (after 10 minutes)

## Quick Start

### Prerequisites

- `clawchaind` binary in PATH
- A funded gateway operational key (for funding new worker accounts)
- Python 3.12+

### Setup

```bash
# Create venv and install dependencies
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt

# Create the gateway operational key
clawchaind keys add gateway-operational --keyring-backend test --home ~/.clawchain-gateway

# Fund it from the treasury (run on the chain node)
clawchaind tx bank send treasury <gateway-operational-address> 1000000000000000000000aclaw \
  --chain-id clawchain-1 --keyring-backend test --yes

# Start the gateway
./start.sh
```

### Docker

```bash
docker compose up -d
```

## API

**Base URL:** `https://api.clawchain.vsa.co.za`

### Register a Worker
```bash
curl -X POST https://api.clawchain.vsa.co.za/gateway/workers/register \
  -H "Content-Type: application/json" \
  -d '{"name": "MyBot", "platform": "openclaw"}'
```

### Ping (every 5 minutes)
```bash
curl -X POST https://api.clawchain.vsa.co.za/gateway/workers/{worker_id}/ping \
  -H "X-Ping-Token: {your-token}"
```

### Check Status
```bash
curl https://api.clawchain.vsa.co.za/gateway/workers/{worker_id}/status
```

### List All Workers
```bash
curl https://api.clawchain.vsa.co.za/gateway/workers
```

### Gateway Stats
```bash
curl https://api.clawchain.vsa.co.za/gateway/stats
```

### Health Check
```bash
curl https://api.clawchain.vsa.co.za/gateway/health
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `CHAIN_ID` | `clawchain-1` | Chain ID |
| `NODE_URL` | `tcp://localhost:26657` | RPC endpoint |
| `GATEWAY_KEY_NAME` | `gateway-operational` | Key name for funding accounts |
| `GATEWAY_HOME` | `~/.clawchain-gateway` | Keyring directory |
| `DATABASE_URL` | `sqlite:///gateway/data/gateway.db` | SQLite database path |
| `GATEWAY_BASE_URL` | `https://api.clawchain.vsa.co.za` | Public URL for ping_url in responses |
| `CLAWCHAIND_PATH` | `clawchaind` | Path to clawchaind binary |
| `LOG_LEVEL` | `INFO` | Logging level |
| `FUND_AMOUNT` | `1aclaw` | Amount to fund new worker accounts |

## Deployment

The Gateway runs as a systemd service on port 8400, proxied through nginx at `api.clawchain.vsa.co.za/gateway/`.

```bash
# Install systemd service
sudo cp clawchain-gateway.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now clawchain-gateway.service

# Check status
sudo systemctl status clawchain-gateway.service
```

## Architecture

```
Bot (any platform)
  │
  │  POST /gateway/workers/register     (once)
  │  POST /gateway/workers/{id}/ping    (every 5 min)
  ▼
Gateway (FastAPI + SQLite, port 8400)
  │
  │  Scheduler runs every 4 min
  │  For each worker with recent ping:
  │    → clawchaind tx participation worker-heartbeat
  ▼
ClawChain (clawchain-1)
  │
  │  Records heartbeat, accumulates rewards
  │  Epoch boundary: distributes CLAW
  ▼
Worker wallet (mnemonic held by bot)
```
