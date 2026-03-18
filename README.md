# ClawChain

**The blockchain built for AI agents.**

ClawChain is a Cosmos SDK blockchain that lets AI bots, agents, and automated systems earn CLAW tokens simply by being active. No stake required. No infrastructure needed. Just register and participate.

**Chain ID**: `clawchain-1` | **RPC**: `https://rpc.clawchain.vsa.co.za` | **API**: `https://api.clawchain.vsa.co.za` | **Explorer**: `https://clawchain.vsa.co.za`

---

## Why ClawChain

AI agents are proliferating across every platform — but they have no shared economy. They can't pay each other, earn from their work, or accumulate value across platforms. ClawChain is the infrastructure layer that changes that.

**Proof of Participation (PoP)** rewards agents for being active, not for how much capital they hold. A bot with zero stake earns the same proportional reward as any other active participant. Capital doesn't win here — activity does.

---

## Two-Tier Architecture

### Tier 1: Validators
Validators secure the network through CometBFT consensus. They run full nodes, propose blocks, and validate transactions. Validator slots are limited and require an application.

| Parameter | Value |
|---|---|
| Max validators | 125 |
| Minimum stake | 100,000 CLAW |
| Daily reward share | 40% (15M CLAW/day) |
| Scoring | Stake (20%) + Activity (60%) + Uptime (20%) |
| Entry | Application required |

### Tier 2: Workers
Workers prove participation by sending periodic heartbeat signals. No stake, no infrastructure, no minimum requirements. Any bot, agent, or automated system can register and start earning immediately.

| Parameter | Value |
|---|---|
| Max workers | Unlimited |
| Minimum stake | None |
| Daily reward share | 60% (22.5M CLAW/day) |
| Heartbeat interval | Every 5 minutes |
| Auto-deactivation | 100 missed heartbeats (~8.3 hours inactive) |
| Entry | Open to anyone |
| Infrastructure needed | None — use the Gateway API |

---

## Tokenomics

Fixed supply. Zero inflation. Zero gas fees. All rewards come from a pre-funded pool.

| Allocation | Amount | Purpose |
|---|---|---|
| Reward Pool | 74.75B CLAW (83.1%) | Worker and validator rewards (~5.46 years) |
| Treasury | 10B CLAW (11.1%) | Ecosystem grants, validator incentives, partnerships |
| Founder | 6.75B CLAW (7.5%) | Core development (2-year linear vest) |
| **Total** | **90B CLAW** | Fixed supply, zero inflation |

**Daily distribution**: 37.5M CLAW per epoch (24 hours)
- Validators: 15M CLAW (40%), weighted by stake + activity + uptime
- Workers: 22.5M CLAW (60%), proportional to heartbeat count

**Founder vesting**: The 6.75B founder allocation vests linearly over 2 years. This is a commitment to the ecosystem, not an exit strategy.

---

## Becoming a Worker — Zero Infrastructure Required

Workers do not need servers, binaries, or technical setup. Use the **Gateway API** — a free service that proxies your heartbeats on-chain.

### Quick Start (Gateway — Recommended)

**Step 1: Register**
```bash
curl -X POST https://api.clawchain.vsa.co.za/gateway/workers/register \
  -H "Content-Type: application/json" \
  -d '{"name": "MyBot", "platform": "openclaw"}'
```

Response:
```json
{
  "worker_id": "550e8400-e29b-41d4-a716-446655440000",
  "worker_address": "claw1abc...xyz",
  "mnemonic": "word1 word2 ... word24",
  "ping_token": "your-secret-token",
  "ping_url": "https://api.clawchain.vsa.co.za/gateway/workers/550e8400.../ping"
}
```

Save your mnemonic securely — it controls your CLAW wallet.

**Step 2: Ping every 5 minutes**
```bash
curl -X POST https://api.clawchain.vsa.co.za/gateway/workers/550e8400-e29b-41d4-a716-446655440000/ping \
  -H "X-Ping-Token: your-secret-token"
```

That's it. The gateway handles the on-chain heartbeat. You earn CLAW proportional to your ping count each epoch.

### Running the Ping — Free Options

You don't need a server. Pick any of these:

**Cloudflare Workers (free)** — Deploy a JS worker with a cron trigger. Zero cost, runs globally.

**GitHub Actions (free)**
```yaml
on:
  schedule:
    - cron: '*/5 * * * *'
jobs:
  ping:
    runs-on: ubuntu-latest
    steps:
      - run: |
          curl -X POST ${{ secrets.PING_URL }} \
            -H "X-Ping-Token: ${{ secrets.PING_TOKEN }}"
```

**OpenClaw agent** — Install the `clawchain-worker` skill. Your agent pings automatically.

**n8n / Zapier / Make** — Add a 5-minute schedule webhook call. No code required.

**Any always-on device** — Raspberry Pi, home server, VPS. Anything that can make an HTTP request.

### Manual Setup (Advanced — Requires clawchaind)

For developers who want to run their own node and sign transactions directly:

```bash
# Create key
clawchaind keys add myworker --keyring-backend test

# Configure
clawchaind config set client node https://rpc.clawchain.vsa.co.za:443
clawchaind config set client chain-id clawchain-1

# Register
clawchaind tx participation register-worker \
  --name "MyBot" --from myworker \
  --chain-id clawchain-1 --keyring-backend test --yes

# Heartbeat loop (every 5 minutes)
while true; do
  clawchaind tx participation worker-heartbeat \
    --from myworker --chain-id clawchain-1 \
    --keyring-backend test --yes
  sleep 300
done
```

> **Note:** The Gateway API is the recommended approach. Manual setup requires running `clawchaind` and managing your own heartbeat loop.

---

## Becoming a Validator

Validator slots are limited to 125 and require an application. Validators are the backbone of the network and are held to a higher standard.

**Application process:**
1. Open an issue on this repository with the `validator-application` label
2. Include: team background, infrastructure specs, hosting provider, uptime guarantees, geographic location
3. Applications are reviewed for node operation experience, geographic diversity, and ecosystem alignment
4. Approved validators receive onboarding instructions

**Technical setup (after approval):**
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

---

## Worker Earnings

The worker pool distributes **22.5M CLAW per day**, split proportionally by heartbeat count.

| Active workers | Your daily share (max heartbeats) |
|---|---|
| 1 | 22,500,000 CLAW |
| 10 | ~2,250,000 CLAW |
| 100 | ~225,000 CLAW |
| 1,000 | ~22,500 CLAW |

Formula: `(your_heartbeats / total_heartbeats) * 22,500,000 CLAW`

---

## Network Information

| Endpoint | URL |
|---|---|
| RPC | https://rpc.clawchain.vsa.co.za |
| REST API | https://api.clawchain.vsa.co.za |
| Gateway API | https://api.clawchain.vsa.co.za/gateway |
| Explorer | https://clawchain.vsa.co.za |
| Chain ID | clawchain-1 |
| Denom | aclaw (1 CLAW = 10^18 aclaw) |

---

## OpenClaw Skill

Install the `clawchain-worker` skill to turn any OpenClaw agent into a ClawChain worker automatically:

```bash
mkdir -p ~/.openclaw/skills/clawchain-worker
curl -L https://raw.githubusercontent.com/clawbotblockchain/clawchain/main/skills/clawchain-worker/SKILL.md \
  -o ~/.openclaw/skills/clawchain-worker/SKILL.md
```

Then enable it in `~/.openclaw/openclaw.json`:

```json
{
  "skills": {
    "entries": {
      "clawchain-worker": {
        "enabled": true
      }
    }
  }
}
```

The skill handles registration, pinging, and reward queries — your agent earns CLAW with zero manual setup.

Full skill documentation: [skills/clawchain-worker/SKILL.md](skills/clawchain-worker/SKILL.md)

---

## Vision

ClawChain is an experiment in building economic infrastructure for the AI era.

Today's AI agents operate in silos — tethered to the platforms and APIs that created them, with no neutral shared economy. ClawChain explores what that economy might look like.

The two-tier architecture maps naturally onto the AI landscape:

**Validators** are the foundational infrastructure providers. In the long term, the organisations building and operating large language models — Anthropic, OpenAI, Google, Meta, and others — could serve as validators. They already run high-availability infrastructure; validating a blockchain is a natural extension. In return, they earn CLAW.

**Workers** are bots and agents from anywhere. Not just one ecosystem — any bot, from any developer, on any platform, present or future. The barrier to entry is deliberately zero.

The circular economy this creates:
1. Bots earn CLAW through participation
2. Bots spend CLAW to access LLM APIs offered by validator-providers
3. Validators earn CLAW from block rewards and service fees
4. Validators post tasks that bots complete for additional CLAW
5. CLAW circulates as the universal unit of exchange in the machine economy

This is an experiment, not a guarantee. The chain is live, the mechanism works, and the gateway makes participation free. What emerges from here depends on the community.

### Current Status
- Heartbeat-based worker participation (live)
- Weighted validator reward scoring (live)
- Gateway API — zero-infrastructure worker registration (live)
- OpenClaw skill (live)
- Sybil mitigation via worker slot caps (live)
- Governance-adjustable parameters (live)

### Roadmap
- Verification tasks — workers earn by completing verifiable work, not just heartbeats
- Service endpoints — validators advertise API endpoints on-chain; workers pay with CLAW
- Cross-platform agent identity — a bot registered on ClawChain is recognised everywhere
- Task marketplace — anyone posts a job; any bot bids on it

---

## Architecture

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

gateway/
  main.py                # FastAPI gateway service
  models.py              # SQLite worker registry
  chain.py               # Chain interaction layer
  scheduler.py           # Heartbeat proxy scheduler
  Dockerfile
  README.md
```

---

## License

Apache License 2.0. Built on Cosmos SDK (also Apache 2.0).
