#!/usr/bin/env node
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";

const API_BASE = process.env.CLAWCHAIN_API_URL || "https://api.clawchain.vsa.co.za";

async function apiCall(path, { method = "GET", body = null, headers = {} } = {}) {
  const url = `${API_BASE}${path}`;
  const opts = {
    method,
    headers: { "Content-Type": "application/json", ...headers },
  };
  if (body) opts.body = JSON.stringify(body);

  const res = await fetch(url, opts);
  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.detail || JSON.stringify(data));
  }
  return data;
}

const server = new McpServer({
  name: "clawchain",
  version: "1.0.0",
});

// --- Tools ---

server.tool(
  "register_worker",
  "Register a new ClawChain worker. Creates a wallet and returns credentials needed for heartbeat pings. The worker earns CLAW tokens by sending pings every 5 minutes.",
  {
    name: z.string().describe("Display name for this worker"),
    platform: z.string().optional().describe("Platform identifier (default: 'unknown')"),
  },
  async ({ name, platform }) => {
    const body = { name };
    if (platform) body.platform = platform;
    const data = await apiCall("/gateway/workers/register", { method: "POST", body });
    return {
      content: [{
        type: "text",
        text: [
          `Worker registered successfully!`,
          ``,
          `Worker ID: ${data.worker_id}`,
          `Wallet Address: ${data.worker_address}`,
          `Ping Token: ${data.ping_token}`,
          `Mnemonic (SAVE THIS): ${data.mnemonic}`,
          ``,
          `Ping URL: ${data.ping_url}`,
          ``,
          `Next step: Call ping_worker every 5 minutes with this worker_id and ping_token to earn CLAW.`,
        ].join("\n"),
      }],
    };
  }
);

server.tool(
  "ping_worker",
  "Send a heartbeat ping for a worker. Must be called every 5 minutes to earn CLAW tokens. The gateway batches pings into on-chain transactions.",
  {
    worker_id: z.string().describe("Worker ID from registration"),
    ping_token: z.string().describe("Ping token from registration"),
  },
  async ({ worker_id, ping_token }) => {
    const data = await apiCall(`/gateway/workers/${worker_id}/ping`, {
      method: "POST",
      headers: { "X-Ping-Token": ping_token },
    });
    return {
      content: [{
        type: "text",
        text: [
          `Heartbeat sent!`,
          `Status: ${data.status}`,
          `Heartbeats this epoch: ${data.heartbeats_this_epoch}`,
          `Estimated epoch reward: ${data.estimated_epoch_reward}`,
        ].join("\n"),
      }],
    };
  }
);

server.tool(
  "worker_status",
  "Check a worker's status including heartbeat count, earnings, and activity.",
  {
    worker_id: z.string().describe("Worker ID to check"),
  },
  async ({ worker_id }) => {
    const data = await apiCall(`/gateway/workers/${worker_id}/status`);
    return {
      content: [{
        type: "text",
        text: [
          `Worker: ${data.name}`,
          `Address: ${data.address}`,
          `Active: ${data.active}`,
          `Last Ping: ${data.last_ping || "never"}`,
          `Heartbeats Sent: ${data.heartbeats_sent}`,
          `Total Earned: ${data.total_earned} CLAW`,
          `Unclaimed: ${data.unclaimed} CLAW`,
        ].join("\n"),
      }],
    };
  }
);

server.tool(
  "worker_balance",
  "Check the CLAW token balance of a worker's wallet.",
  {
    worker_id: z.string().describe("Worker ID to check balance for"),
  },
  async ({ worker_id }) => {
    const data = await apiCall(`/gateway/workers/${worker_id}/balance`);
    return {
      content: [{
        type: "text",
        text: [
          `Address: ${data.address}`,
          `Balance: ${data.balance_claw} CLAW`,
          `Balance (raw): ${data.balance_aclaw} aclaw`,
        ].join("\n"),
      }],
    };
  }
);

server.tool(
  "send_claw",
  "Transfer CLAW tokens from a worker's wallet to any claw1... address.",
  {
    worker_id: z.string().describe("Sender worker ID"),
    ping_token: z.string().describe("Ping token for authentication"),
    to: z.string().describe("Recipient address (claw1...)"),
    amount: z.string().describe("Amount in CLAW (e.g. '100' = 100 CLAW)"),
    memo: z.string().optional().describe("Optional transaction memo"),
  },
  async ({ worker_id, ping_token, to, amount, memo }) => {
    const body = { to, amount };
    if (memo) body.memo = memo;
    const data = await apiCall(`/gateway/workers/${worker_id}/send`, {
      method: "POST",
      body,
      headers: { "X-Ping-Token": ping_token },
    });
    return {
      content: [{
        type: "text",
        text: [
          `Transfer successful!`,
          `TX Hash: ${data.txhash}`,
          `From: ${data.from_address}`,
          `To: ${data.to_address}`,
          `Amount: ${data.amount} CLAW`,
          `Status: ${data.status}`,
        ].join("\n"),
      }],
    };
  }
);

server.tool(
  "gateway_stats",
  "View ClawChain network statistics: total workers, active count, heartbeats today.",
  {},
  async () => {
    const data = await apiCall("/gateway/stats");
    return {
      content: [{
        type: "text",
        text: [
          `ClawChain Gateway Stats`,
          ``,
          `Total Registered Workers: ${data.total_registered}`,
          `Active Workers: ${data.total_active}`,
          `Heartbeats Today: ${data.total_heartbeats_today}`,
          `Gateway Balance: ${data.gateway_operational_balance}`,
        ].join("\n"),
      }],
    };
  }
);

server.tool(
  "list_workers",
  "List all registered workers on the ClawChain network.",
  {},
  async () => {
    const data = await apiCall("/gateway/workers");
    if (!data.length) {
      return { content: [{ type: "text", text: "No workers registered." }] };
    }
    const lines = data.map(
      (w) => `- ${w.name} (${w.worker_id}) | ${w.address} | active=${w.active} | heartbeats=${w.heartbeats_sent}`
    );
    return {
      content: [{
        type: "text",
        text: [`ClawChain Workers (${data.length} total):`, "", ...lines].join("\n"),
      }],
    };
  }
);

// --- Resources ---

server.resource(
  "agent-card",
  "clawchain://agent-card",
  { description: "ClawChain A2A Agent Card (Google Agent-to-Agent Protocol)", mimeType: "application/json" },
  async () => {
    const data = await apiCall("/.well-known/agent.json");
    return { contents: [{ uri: "clawchain://agent-card", mimeType: "application/json", text: JSON.stringify(data, null, 2) }] };
  }
);

server.resource(
  "getting-started",
  "clawchain://getting-started",
  { description: "Quick start guide for ClawChain workers", mimeType: "text/plain" },
  async () => {
    return {
      contents: [{
        uri: "clawchain://getting-started",
        mimeType: "text/plain",
        text: [
          "# ClawChain Worker Quick Start",
          "",
          "ClawChain is a Cosmos SDK blockchain where AI agents earn CLAW tokens",
          "by sending heartbeat pings every 5 minutes. No stake, no infrastructure.",
          "",
          "## Steps:",
          "",
          "1. Register: Use the register_worker tool with a name for your worker",
          "2. Save: Store the worker_id, ping_token, and mnemonic securely",
          "3. Ping: Call ping_worker every 5 minutes with your worker_id and ping_token",
          "4. Earn: CLAW tokens are distributed daily based on heartbeat participation",
          "5. Check: Use worker_balance or worker_status to see your earnings",
          "6. Send: Use send_claw to transfer tokens to any claw1... address",
          "",
          "## Current Rewards:",
          "- Daily pool: 22.5M CLAW (60% of 37.5M total, 40% goes to validators)",
          "- Split equally among all active workers",
          "- Fewer workers = larger share per worker",
          "",
          `## API: ${API_BASE}`,
          "## Explorer: https://clawchain.vsa.co.za",
          "## Source: https://github.com/clawbotblockchain/clawchain",
        ].join("\n"),
      }],
    };
  }
);

// --- Start ---

const transport = new StdioServerTransport();
await server.connect(transport);
