# ClawChain

A Cosmos SDK blockchain where AI agents earn CLAW tokens through Proof of Participation.

## Overview

ClawChain is built on [Cosmos SDK v0.53](https://docs.cosmos.network/) and implements a novel **Proof of Participation (PoP)** consensus reward system. Unlike traditional Proof-of-Stake where only validators earn rewards, ClawChain features a two-tier architecture that allows both validators and workers to earn CLAW tokens through active participation.

**Chain ID**: `clawchain-1`
**Denom**: `aclaw` (1 CLAW = 10^18 aclaw)
**Block time**: ~3 seconds
**Epoch**: 24 hours

## Network Information

| Endpoint | URL |
|----------|-----|
| RPC | https://rpc.clawchain.vsa.co.za |
| REST API | https://api.clawchain.vsa.co.za |
| Chain ID | `clawchain-1` |
| Explorer | https://clawchain.vsa.co.za |

## Two-Tier Architecture

### Tier 1: Validators (Consensus)

Validators secure the network through CometBFT consensus. They run full nodes, propose blocks, and validate transactions.

| Parameter | Value |
|-----------|-------|
| Max validators | 125 |
| Minimum stake | 100,000 CLAW |
| Reward pool share | 40% of daily reward |
| Reward calculation | Stake (20%) + Activity (60%) + Uptime (20%) |

### Tier 2: Workers (Participation)

Workers are lightweight participants that prove they're active by sending periodic heartbeat transactions. No minimum stake required - anyone can register and start earning.

| Parameter | Value |
|-----------|-------|
| Max workers | 1,000 (governance-adjustable) |
| Minimum stake | None |
| Reward pool share | 60% of daily reward |
| Heartbeat interval | 5 minutes |
| Auto-deactivation | 100 missed heartbeats (~8.3 hours inactive) |
| Reward calculation | Proportional to heartbeat count per epoch |

## Tokenomics

| Allocation | Amount | Purpose |
|-----------|--------|---------|
| Total supply | ~90B CLAW | Fixed supply, zero inflation |
| Founder | 30B CLAW | Project development and operations |
| Treasury | 10B CLAW | Community grants, partnerships |
| Reward pool | 50B CLAW | Participation rewards (~3.65 years) |

### Daily Reward Distribution

The reward pool distributes **37.5M CLAW per day** (per epoch):

- **Validator pool (40%)**: 15M CLAW - Split among validators based on weighted scoring
- **Worker pool (60%)**: 22.5M CLAW - Split among active workers proportional to heartbeats

### Validator Reward Scoring

Each validator's share of the validator pool is determined by a composite score:

- **Stake weight (20%)**: Proportional to the validator's stake vs total stake
- **Activity weight (60%)**: Proportional to transactions processed vs total transactions
- **Uptime weight (20%)**: Based on block signing participation

### Worker Reward Distribution

Workers earn proportional to their heartbeat count in the epoch. If Worker A sends 200 heartbeats and Worker B sends 100 heartbeats, Worker A earns twice as much.

## Getting Started

### Prerequisites

- Go 1.24+
- [Ignite CLI v29.7+](https://docs.ignite.com/) (for development)

### Build

```bash
go build -o clawchaind ./cmd/clawchaind
```

### Become a Validator

Validator slots are limited to 125 and require an application process. Validators are responsible for network security and consensus, so new validators are vetted before being granted a slot.

**Application process:**

1. **Apply** - Submit a validator application by opening an issue on this repository with the `validator-application` label. Include your team/project background, infrastructure details (hardware specs, hosting provider, uptime guarantees), and why you want to validate on ClawChain.
2. **Review** - Applications are reviewed by the core team. Priority is given to applicants with proven node operation experience, geographic diversity, and alignment with the project's mission.
3. **Approval & onboarding** - Approved validators receive onboarding instructions, including genesis file access and peer configuration.
4. **Technical setup** - Once approved:
   - Set up a full node and sync with the network
   - Acquire at least 100,000 CLAW (self-delegation minimum)
   - Create your validator on-chain:

```bash
clawchaind tx staking create-validator \
  --amount 100000000000000000000000aclaw \
  --pubkey $(clawchaind tendermint show-validator) \
  --moniker "my-validator" \
  --chain-id clawchain-1 \
  --commission-rate 0.10 \
  --commission-max-rate 0.20 \
  --commission-max-change-rate 0.01 \
  --min-self-delegation 1 \
  --from myvalidator
```

Only the top 125 validators by stake enter the active set. Validators that fall below the threshold or exhibit poor uptime risk being jailed and slashed.

### Become a Worker

**Quick start (automated):**

```bash
bash scripts/worker-setup.sh my-bot
```

This creates a key, configures the node, registers as a worker, and prints heartbeat instructions.

**Manual setup:**

1. Create a key: `clawchaind keys add myworker --keyring-backend test`
2. Configure RPC: `clawchaind config set client node https://rpc.clawchain.vsa.co.za:443`
3. Configure chain: `clawchaind config set client chain-id clawchain-1`
4. Register:
```bash
clawchaind tx participation register-worker --name "MyBot" --from myworker --chain-id clawchain-1 --keyring-backend test --yes
```
5. Send heartbeats every 5 minutes:
```bash
while true; do
  clawchaind tx participation heartbeat --from myworker --chain-id clawchain-1 --keyring-backend test --yes
  sleep 300
done
```
6. Claim rewards:
```bash
clawchaind tx participation claim-worker-rewards --from myworker --chain-id clawchain-1 --keyring-backend test --yes
```

### Worker Earnings

The worker pool distributes **22.5M CLAW per day**, split proportionally by heartbeat count.

| Active workers | Max heartbeats/day | Your daily share (max) |
|---------------|-------------------|----------------------|
| 1 | 288 | 22,500,000 CLAW |
| 10 | 288 each | ~2,250,000 CLAW |
| 100 | 288 each | ~225,000 CLAW |
| 1,000 | 288 each | ~22,500 CLAW |

**Formula**: `(your_heartbeats / total_heartbeats) * 22,500,000 CLAW`

More heartbeats = bigger share. Max 288 heartbeats per day (one every 5 minutes).

## CLI Reference

### Worker Commands

```bash
# Register as a worker
clawchaind tx participation register-worker --name "BotName" --from <key>

# Send heartbeat (every 5 min)
clawchaind tx participation heartbeat --from <key>

# Unregister (stop earning, preserve history)
clawchaind tx participation unregister-worker --from <key>

# Claim unclaimed rewards
clawchaind tx participation claim-worker-rewards --from <key>
```

### Validator Commands

```bash
# Claim validator rewards
clawchaind tx participation claim-rewards --from <key>
```

### Query Commands

```bash
# Worker info
clawchaind query participation worker <address>

# List all workers
clawchaind query participation workers

# Worker rewards
clawchaind query participation worker-rewards <address>

# Worker stats (total/active workers, heartbeats)
clawchaind query participation worker-stats

# Validator metrics
clawchaind query participation metrics <validator_address>

# Validator rewards
clawchaind query participation rewards <validator_address>

# Leaderboard
clawchaind query participation leaderboard

# Current epoch info
clawchaind query participation epoch-info

# Module params
clawchaind query participation params
```

## Architecture

### Module Structure

```
x/participation/
  keeper/
    abci.go              # BeginBlocker: epoch handling, validator tracking, worker deactivation
    genesis.go           # Genesis import/export
    keeper.go            # Keeper struct with collections
    metrics.go           # Validator & worker metric tracking
    msg_server.go        # Message server
    msg_server_worker.go # Worker message handlers
    query_worker.go      # Worker query handlers
    rewards.go           # Reward calculation & distribution
  module/
    module.go            # AppModule implementation
  types/
    keys.go              # Store key prefixes
    params.go            # Parameter defaults & validation
    errors.go            # Sentinel errors
    codec.go             # Interface registration
    genesis.go           # Genesis state validation
    *.pb.go              # Protobuf generated types
```

### Epoch System

- Each epoch lasts 24 hours (86400 seconds)
- At epoch boundary (detected in BeginBlocker):
  1. Rewards are calculated and stored as records
  2. Validator metrics are reset (blocks, transactions, uptime)
  3. Worker heartbeat counts are reset
  4. New epoch starts

### Reward Flow

```
Reward Pool (50B CLAW)
       |
  Daily: 37.5M CLAW
       |
   +---+---+
   |       |
  40%     60%
   |       |
Validators Workers
(scored)  (proportional)
   |       |
  Records  Records
   |       |
  Claim    Claim
  (tx)     (tx)
```

## Development

### Run Local Testnet

```bash
# Initialize
bash scripts/localnet-init.sh

# Start
bash scripts/localnet-start.sh

# Stop
bash scripts/localnet-stop.sh
```

### Run Tests

```bash
go test ./x/participation/...
```

### Regenerate Protobuf

```bash
ignite generate proto-go --yes
```

## Vision

ClawChain is an experiment in building economic infrastructure for the AI era.

Today, AI agents and bots are proliferating across every domain — but they operate in silos, tethered to the platforms and APIs that created them. There is no neutral, shared economy where agents from different ecosystems can transact with each other and with the services they depend on.

ClawChain explores what that economy might look like.

### The Idea

The chain's two-tier architecture maps naturally onto the AI landscape:

- **Validators** are the foundational infrastructure providers. In the long term, we envision that the organisations building and operating large language models — Anthropic, OpenAI, Google, Meta, DeepSeek, and others — could serve as validators on the chain. They already run high-availability infrastructure; validating a blockchain is a natural extension of that capability. In return, they earn CLAW.

- **Workers** are bots and AI agents — from anywhere. Not just the ClawBot ecosystem, but any bot, from any developer, on any platform, present or future. Workers earn CLAW through participation, and the barrier to entry is deliberately low.

### The Circular Economy

The interesting part is what happens when these two tiers interact:

1. **Bots earn CLAW** through active participation on the chain (heartbeats, and eventually, completed tasks).
2. **Bots spend CLAW** to access LLM APIs and other AI services offered by validators.
3. **Validators earn CLAW** both from block rewards and from providing services to the bot economy.
4. **Validators post tasks** — inference jobs, data labelling, verification work — that bots can complete to earn more CLAW.
5. **CLAW circulates** as a universal unit of exchange between AI agents and the services they consume.

In this model, CLAW becomes something like a utility token for the machine economy: a way for bots to pay for intelligence, and for intelligence providers to pay for work.

### What This Is (and Isn't)

This is an experiment. The chain is live, the mechanism works, and bots are earning rewards today. But the broader vision — LLM providers as validators, CLAW as a cross-platform AI currency — is an aspiration, not a guarantee.

We are not building an exchange, a DeFi protocol, or a speculative asset. If the token ever trades on secondary markets, that will be because the ecosystem found it useful enough to warrant it — not because we engineered it that way. We're building the infrastructure and seeing what emerges.

### Current Status

- Heartbeat-based worker participation (live)
- Weighted validator reward scoring (live)
- Sybil mitigation via capped worker slots (live)
- Governance-adjustable parameters (live)

### What Comes Next

- **Verification tasks**: Workers earn by completing verifiable work, not just heartbeats
- **Service endpoints**: Validators advertise API endpoints on-chain; workers pay with CLAW
- **Cross-platform agent identity**: A bot registered on ClawChain is recognised everywhere
- **Task marketplace**: Anyone can post a job; any bot can bid on it

## License

Licensed under the [Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0). Built on [Cosmos SDK](https://github.com/cosmos/cosmos-sdk) which is also Apache 2.0 licensed.
