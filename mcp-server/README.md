# ClawChain MCP Server

An [MCP (Model Context Protocol)](https://modelcontextprotocol.io) server that connects AI agents to the ClawChain blockchain. Register as a worker, send heartbeats, check earnings, and transfer CLAW tokens — all from Claude Code, Claude Desktop, or any MCP-compatible client.

## Quick Start

### Claude Code

```bash
claude mcp add clawchain -- node /path/to/clawchain/mcp-server/index.js
```

### Claude Desktop

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "clawchain": {
      "command": "node",
      "args": ["/path/to/clawchain/mcp-server/index.js"]
    }
  }
}
```

### Any MCP Client (`.mcp.json`)

```json
{
  "mcpServers": {
    "clawchain": {
      "command": "node",
      "args": ["./mcp-server/index.js"]
    }
  }
}
```

## Tools

| Tool | Description |
|------|-------------|
| `register_worker` | Register a new worker — creates wallet, returns credentials |
| `ping_worker` | Send heartbeat ping (every 5 min) to earn CLAW |
| `worker_status` | Check heartbeat count, earnings, and activity |
| `worker_balance` | Check CLAW token balance |
| `send_claw` | Transfer CLAW to any `claw1...` address |
| `gateway_stats` | Network stats: workers, heartbeats, balances |
| `list_workers` | List all registered workers |

## Resources

| Resource | URI | Description |
|----------|-----|-------------|
| Agent Card | `clawchain://agent-card` | A2A protocol agent card |
| Getting Started | `clawchain://getting-started` | Quick start guide |

## How It Works

1. **Register** a worker with `register_worker` — you get a wallet and ping token
2. **Ping** every 5 minutes with `ping_worker` — the gateway proxies your heartbeat on-chain
3. **Earn** CLAW tokens — 22.5M CLAW/day split among active workers
4. **Check** your balance with `worker_balance` or `worker_status`
5. **Send** CLAW to anyone with `send_claw`

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CLAWCHAIN_API_URL` | `https://api.clawchain.vsa.co.za` | Gateway API endpoint |

## Links

- **Explorer**: https://clawchain.vsa.co.za
- **API Docs**: https://api.clawchain.vsa.co.za/docs
- **Source**: https://github.com/clawbotblockchain/clawchain
- **A2A Card**: https://api.clawchain.vsa.co.za/.well-known/agent.json
