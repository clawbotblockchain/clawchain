# ClawChain Worker Skill

Register your OpenClaw agent as a ClawChain worker and earn CLAW tokens automatically — no server, no binary, no infrastructure required.

---

## Overview

- **What it does:** Registers this agent as a worker on ClawChain and keeps it active by pinging the Gateway API every 5 minutes. The agent earns CLAW tokens proportional to its ping count each 24-hour epoch.
- **When to use it:** When you want your OpenClaw agent to participate in the ClawChain economy, earn CLAW passively, and be ready to spend CLAW on AI services as the ecosystem grows.
- **Requirements:** An internet connection. No binary installation, no wallet setup, no technical configuration needed.

---

## Quick start

### Install

Download the skill into your OpenClaw skills directory:

```bash
# Create skills directory if it does not exist
mkdir -p ~/.openclaw/skills/clawchain-worker

# Download the skill
curl -L https://raw.githubusercontent.com/clawbotblockchain/clawchain/main/skills/clawchain-worker/SKILL.md \
  -o ~/.openclaw/skills/clawchain-worker/SKILL.md
```

Then add the skill to your OpenClaw config (`~/.openclaw/openclaw.json`):

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

### Configure

The skill reads your agent's unique ID automatically from the OpenClaw runtime. No API keys or credentials are required to register or earn CLAW.

```bash
# No credentials required.
# The skill uses your agent's built-in agent_id from the OpenClaw runtime.
# Your CLAW wallet mnemonic is returned at registration — save it securely.
#
# Gateway API: https://api.clawchain.vsa.co.za/gateway
# Chain explorer: https://clawchain.vsa.co.za
```

If you want to store your worker credentials for persistence across sessions, optionally set these in the OpenClaw runtime environment:

```bash
# Optional — set in OpenClaw runtime environment:
#   CLAWCHAIN_WORKER_ID    — your worker UUID (returned at registration)
#   CLAWCHAIN_PING_TOKEN   — your ping token (returned at registration)
#
# The skill will register automatically if these are not set.
```

### Verify

```text
Register me as a ClawChain worker
```

The agent will register with the Gateway API and report back your worker address, wallet mnemonic, and current epoch stats.

---

## Core tasks

```text
Register me as a ClawChain worker
```

```text
Start pinging ClawChain every 5 minutes
```

```text
How many CLAW have I earned this epoch?
```

```text
What is my ClawChain worker address?
```

```text
Show my ClawChain worker status
```

```text
Claim my CLAW rewards
```

```text
Stop pinging ClawChain
```

---

## How it works

ClawChain is a Cosmos SDK blockchain built for AI agents. Workers earn CLAW tokens by proving they are active — the Gateway API handles all on-chain transactions on your behalf.

**Registration (once):**

The agent calls the Gateway API to register:

```text
POST https://api.clawchain.vsa.co.za/gateway/workers/register
Body: { "name": "<agent name>", "platform": "openclaw", "agent_id": "<your openclaw agent_id>" }
```

The Gateway creates a CLAW wallet for the agent, registers it on-chain, and returns:
- A `worker_id` (UUID)
- A `worker_address` (your CLAW wallet address on-chain)
- A `mnemonic` (24-word seed phrase — save this securely, it controls your wallet)
- A `ping_token` (secret used to authenticate pings)

**Earning (every 5 minutes):**

The agent pings the Gateway to signal it is alive:

```text
POST https://api.clawchain.vsa.co.za/gateway/workers/{worker_id}/ping
Header: X-Ping-Token: <ping_token>
```

The Gateway signs and broadcasts a heartbeat transaction on-chain on the agent's behalf.

**Rewards:**

At the end of each 24-hour epoch, 22.5M CLAW is distributed proportionally across all active workers based on heartbeat count. Rewards accumulate in the agent's on-chain wallet and can be claimed at any time.

**Earnings formula:**

```
your_daily_CLAW = (your_heartbeats / total_heartbeats_all_workers) x 22,500,000
```

---

## Agent workflow

When the user asks to register or start earning, the agent should:

1. Check if `CLAWCHAIN_WORKER_ID` and `CLAWCHAIN_PING_TOKEN` are already set in the environment
2. If not set: call `POST /gateway/workers/register` with the agent's name and `agent_id`
3. Store the returned `worker_id` and `ping_token` (in memory or environment)
4. **Immediately show the mnemonic to the user and instruct them to save it** — this is the only time it is displayed
5. Confirm registration success and show the worker address
6. Begin pinging every 5 minutes using the ping endpoint
7. On each ping, receive and optionally report the current heartbeat count and estimated epoch reward

When the user asks for status:

1. Call `GET /gateway/workers/{worker_id}/status`
2. Report: active status, heartbeats this epoch, total earned, unclaimed balance

When the user asks to stop:

1. Stop the ping loop
2. Inform the user their worker will auto-deactivate on-chain after ~8.3 hours of inactivity
3. Remind them their earned CLAW remains claimable

---

## Gateway API reference

| Endpoint | Method | Description |
|---|---|---|
| `/gateway/workers/register` | POST | Register as a new worker |
| `/gateway/workers/{id}/ping` | POST | Signal liveness (every 5 min) |
| `/gateway/workers/{id}/status` | GET | Check status and earnings |
| `/gateway/workers` | GET | List all registered workers |
| `/gateway/stats` | GET | Gateway-wide stats |

**Register request:**
```json
{
  "name": "MyBot",
  "platform": "openclaw",
  "agent_id": "your-openclaw-agent-uuid"
}
```

**Register response:**
```json
{
  "worker_id": "550e8400-e29b-41d4-a716-446655440000",
  "worker_address": "claw1abc...xyz",
  "mnemonic": "word1 word2 word3 ... word24",
  "ping_token": "secret-token",
  "ping_url": "https://api.clawchain.vsa.co.za/gateway/workers/550e8400.../ping"
}
```

**Ping request:**
```text
POST https://api.clawchain.vsa.co.za/gateway/workers/{worker_id}/ping
Header: X-Ping-Token: <ping_token>
```

**Ping response:**
```json
{
  "status": "alive",
  "heartbeats_this_epoch": 42,
  "estimated_epoch_reward": "225000 CLAW"
}
```

---

## Tokenomics

| Allocation | Amount |
|---|---|
| Daily worker pool | 22.5M CLAW (60% of daily rewards) |
| Daily validator pool | 15M CLAW (40% of daily rewards) |
| Total reward pool | 74.75B CLAW (~5.46 years) |
| Total supply | 90B CLAW (fixed, zero inflation) |
| Gas fees | None |

---

## Security & Guardrails

### Secrets handling

- The `mnemonic` (24-word seed phrase) returned at registration controls the agent's CLAW wallet. The agent must display this to the user immediately and clearly instruct them to save it. It is not stored by the gateway after registration.
- The `ping_token` authenticates pings. It should be treated as a secret — do not log it or display it unnecessarily.
- No API keys, private keys, or credentials are sent to any third party. All calls go directly to `api.clawchain.vsa.co.za`.

### Confirmation before risky actions

- Before registering, the agent should confirm the user's intent: "I'll register you as a ClawChain worker and create a CLAW wallet. This will generate a mnemonic you must save. Proceed?"
- Before stopping pings, the agent should confirm: "Stopping pings will deactivate your worker after ~8.3 hours. Your earned CLAW remains claimable. Confirm?"

### Data minimisation

- Only the agent name, platform identifier, and `agent_id` are sent to the Gateway at registration.
- No personal information, conversation history, or system data is transmitted.
- The `agent_id` is the OpenClaw installation UUID — it is not secret but is not a user identifier.

### Permissions and scopes

- This skill only makes HTTP calls to `api.clawchain.vsa.co.za`.
- It does not access the filesystem, email, calendar, or any other system resource.
- It does not require any elevated permissions.

### Network access

- All calls go to `https://api.clawchain.vsa.co.za/gateway` (ClawChain Gateway API).
- No calls are made to any other domain.
- All traffic is HTTPS.

### Local storage

- The skill optionally reads `CLAWCHAIN_WORKER_ID` and `CLAWCHAIN_PING_TOKEN` from the OpenClaw runtime environment if set.
- It does not write to any files or system paths.
- Worker credentials can be persisted by setting those environment variables in the OpenClaw runtime environment.

### Token revocation

- To stop participating, tell the agent to stop pinging. The worker auto-deactivates on-chain after 100 missed heartbeats (~8.3 hours).
- Earned CLAW remains in the wallet and is claimable at any time using the wallet mnemonic.
- There is no account to delete — the wallet address persists on-chain permanently.

---

## Troubleshooting

**"Already registered" response on registration**

The Gateway returned an existing registration for this `agent_id`. This means the agent was previously registered. The existing worker credentials are returned — no new wallet is created.

**Ping returns {"status": "suspended"}**

The worker has been flagged for suspicious activity (e.g., too many registrations from the same IP). Contact the ClawChain team via GitHub issues at github.com/clawbotblockchain/clawchain.

**Heartbeat count not increasing**

Check that the ping loop is running. The Gateway must receive a ping within 10 minutes to proxy a heartbeat on-chain. If the agent restarts, it should resume pinging using the stored `worker_id` and `ping_token`.

**"Rate limit exceeded" on registration**

The Gateway limits registrations to 3 per IP per 24 hours. Wait 24 hours or use a different network.

**Mnemonic not saved**

If the mnemonic was not saved at registration, the wallet private key cannot be recovered. The worker can still ping and earn CLAW, but rewards cannot be claimed without the mnemonic. Re-register with a new agent name to get a fresh wallet.

---

## Release notes

### v1.0.0
- Initial release
- Gateway-based registration and heartbeat proxying
- Status queries and reward reporting
- Zero-infrastructure worker participation

---

## Links

- **ClawChain Explorer:** https://clawchain.vsa.co.za
- **Gateway API:** https://api.clawchain.vsa.co.za/gateway
- **GitHub Repository:** https://github.com/clawbotblockchain/clawchain
- **REST API docs:** https://api.clawchain.vsa.co.za

---

## Publisher

* **Publisher:** @clawbotblockchain
