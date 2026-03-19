"""ClawChain Gateway — Zero-infrastructure worker heartbeat proxy."""

import os
import pathlib

# Load .env BEFORE any package imports (chain.py reads env at import time)
_GATEWAY_DIR = pathlib.Path(__file__).resolve().parent
_env_path = _GATEWAY_DIR / ".env"
if _env_path.exists():
    with open(_env_path) as _f:
        for _line in _f:
            _line = _line.strip()
            if _line and not _line.startswith("#") and "=" in _line:
                _k, _v = _line.split("=", 1)
                os.environ.setdefault(_k.strip(), _v.strip())

import asyncio
import datetime
import logging

from contextlib import asynccontextmanager
from fastapi import FastAPI, Header, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from pydantic import BaseModel

from . import chain
from .models import Worker, get_engine, get_session_factory, init_db
from .scheduler import heartbeat_loop

logging.basicConfig(
    level=os.getenv("LOG_LEVEL", "INFO"),
    format="%(asctime)s [%(name)s] %(levelname)s: %(message)s",
)
logger = logging.getLogger(__name__)

# Rate limit: minimum seconds between pings from the same worker
MIN_PING_INTERVAL_SECONDS = 30  # 30 seconds

# Database setup — use absolute path relative to gateway dir
_data_dir = _GATEWAY_DIR / "data"
_data_dir.mkdir(exist_ok=True)
DATABASE_URL = os.getenv("DATABASE_URL", f"sqlite:///{_data_dir / 'gateway.db'}")
engine = get_engine(DATABASE_URL)
SessionFactory = get_session_factory(engine)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Initialize database and start background scheduler on startup."""
    init_db(engine)
    chain.init_gateway_home()
    task = asyncio.create_task(heartbeat_loop(SessionFactory))
    logger.info("ClawChain Gateway started")
    yield
    task.cancel()


app = FastAPI(
    title="ClawChain Gateway",
    description="Zero-infrastructure worker heartbeat proxy for ClawChain",
    version="1.0.0",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["GET", "POST", "OPTIONS"],
    allow_headers=["*"],
)


# --- Request/Response models ---

class RegisterRequest(BaseModel):
    name: str
    platform: str = "unknown"


class RegisterResponse(BaseModel):
    worker_id: str
    worker_address: str
    mnemonic: str
    ping_token: str
    ping_url: str


class PingResponse(BaseModel):
    status: str
    heartbeats_this_epoch: int
    estimated_epoch_reward: str


class WorkerStatusResponse(BaseModel):
    name: str
    address: str
    active: bool
    last_ping: str | None
    heartbeats_sent: int
    total_earned: str
    unclaimed: str


class WorkerListItem(BaseModel):
    worker_id: str
    name: str
    address: str
    active: bool
    heartbeats_sent: int


class SendRequest(BaseModel):
    to: str
    amount: str  # Amount in CLAW (human-readable, e.g. "100" = 100 CLAW)
    memo: str = ""


class SendResponse(BaseModel):
    txhash: str
    from_address: str
    to_address: str
    amount: str
    status: str


class BalanceResponse(BaseModel):
    address: str
    balance_claw: str
    balance_aclaw: str


class GatewayStatsResponse(BaseModel):
    total_registered: int
    total_active: int
    total_heartbeats_today: int
    gateway_operational_balance: str


# --- Endpoints ---

@app.post("/gateway/workers/register", response_model=RegisterResponse)
def register_worker(req: RegisterRequest):
    """Register a new worker. Creates a keypair, funds it, and registers on-chain."""
    with SessionFactory() as db:
        # Create the worker record first to get an ID
        worker = Worker(name=req.name, platform=req.platform)
        worker_id = worker.id

        # Create Cosmos keypair
        try:
            key_info = chain.create_worker_key(worker_id)
        except RuntimeError as e:
            raise HTTPException(status_code=500, detail=f"Key creation failed: {e}")

        worker.key_name = key_info["key_name"]
        worker.address = key_info["address"]

        # Fund the account (creates it on-chain)
        fund_tx = chain.fund_account(key_info["address"])
        if not fund_tx:
            logger.warning("Could not fund worker %s — registration may still work with 0 gas", worker_id)

        # Register on-chain
        reg_tx = chain.register_worker(key_info["key_name"], req.name)
        if not reg_tx:
            raise HTTPException(
                status_code=500,
                detail="On-chain registration failed. The gateway operational key may need funding.",
            )

        db.add(worker)
        db.commit()
        db.refresh(worker)

        base_url = os.getenv("GATEWAY_BASE_URL", "https://api.clawchain.vsa.co.za")
        return RegisterResponse(
            worker_id=worker.id,
            worker_address=worker.address,
            mnemonic=key_info["mnemonic"],
            ping_token=worker.ping_token,
            ping_url=f"{base_url}/gateway/workers/{worker.id}/ping",
        )


@app.post("/gateway/workers/{worker_id}/ping", response_model=PingResponse)
def ping_worker(worker_id: str, x_ping_token: str = Header()):
    """Signal that a worker is alive. Must be called every 5 minutes."""
    with SessionFactory() as db:
        worker = db.query(Worker).filter(Worker.id == worker_id).first()
        if not worker:
            raise HTTPException(status_code=404, detail="Worker not found")

        if worker.ping_token != x_ping_token:
            raise HTTPException(status_code=403, detail="Invalid ping token")

        # Rate limit check
        now = datetime.datetime.utcnow()
        if worker.last_ping:
            elapsed = (now - worker.last_ping).total_seconds()
            if elapsed < MIN_PING_INTERVAL_SECONDS:
                raise HTTPException(
                    status_code=429,
                    detail=f"Too soon. Wait {int(MIN_PING_INTERVAL_SECONDS - elapsed)}s before next ping.",
                )

        worker.last_ping = now
        worker.active = True
        db.commit()

        # Query on-chain data for response
        on_chain = chain.query_worker(worker.address)
        heartbeats_this_epoch = 0
        if on_chain and "worker" in on_chain:
            heartbeats_this_epoch = int(on_chain["worker"].get("heartbeat_count", 0))

        # Estimate reward (22.5M CLAW daily pool, proportional to heartbeats)
        stats = chain.query_worker_stats()
        estimated = "0 CLAW"
        if stats:
            total_hb = int(stats.get("total_heartbeats_this_epoch", 0))
            if total_hb > 0 and heartbeats_this_epoch > 0:
                daily_pool = 22_500_000
                share = (heartbeats_this_epoch / total_hb) * daily_pool
                estimated = f"{int(share):,} CLAW"

        return PingResponse(
            status="alive",
            heartbeats_this_epoch=heartbeats_this_epoch,
            estimated_epoch_reward=estimated,
        )


@app.get("/gateway/workers/{worker_id}/status", response_model=WorkerStatusResponse)
def worker_status(worker_id: str):
    """Get status for a specific worker."""
    with SessionFactory() as db:
        worker = db.query(Worker).filter(Worker.id == worker_id).first()
        if not worker:
            raise HTTPException(status_code=404, detail="Worker not found")

        # Query on-chain rewards
        rewards = chain.query_worker_rewards(worker.address)
        total_earned = "0"
        unclaimed = "0"
        if rewards:
            total_earned = rewards.get("total_earned", "0")
            unclaimed = rewards.get("unclaimed", "0")

        return WorkerStatusResponse(
            name=worker.name,
            address=worker.address,
            active=worker.active,
            last_ping=worker.last_ping.isoformat() + "Z" if worker.last_ping else None,
            heartbeats_sent=worker.heartbeats_sent,
            total_earned=f"{total_earned} CLAW",
            unclaimed=f"{unclaimed} CLAW",
        )


@app.post("/gateway/workers/{worker_id}/send", response_model=SendResponse)
def send_claw(worker_id: str, req: SendRequest, x_ping_token: str = Header()):
    """Send CLAW from a worker's wallet to another address."""
    with SessionFactory() as db:
        worker = db.query(Worker).filter(Worker.id == worker_id).first()
        if not worker:
            raise HTTPException(status_code=404, detail="Worker not found")

        if worker.ping_token != x_ping_token:
            raise HTTPException(status_code=403, detail="Invalid ping token")

        # Validate recipient address
        if not req.to.startswith("claw1"):
            raise HTTPException(status_code=400, detail="Invalid recipient address — must start with claw1")

        # Convert human CLAW to aclaw (18 decimals)
        try:
            claw_amount = float(req.amount)
            if claw_amount <= 0:
                raise ValueError()
            aclaw_amount = int(claw_amount * (10**18))
        except (ValueError, OverflowError):
            raise HTTPException(status_code=400, detail="Invalid amount — must be a positive number")

        amount_str = f"{aclaw_amount}aclaw"

        try:
            result = chain.send_tokens(worker.key_name, req.to, amount_str)
        except RuntimeError as e:
            raise HTTPException(status_code=500, detail=f"Transaction failed: {e}")

        if result.get("code", 0) != 0:
            raise HTTPException(status_code=500, detail=f"Transaction rejected by chain (code {result['code']})")

        logger.info("Send: %s → %s (%s CLAW) tx=%s", worker.address, req.to, req.amount, result["txhash"])

        return SendResponse(
            txhash=result["txhash"],
            from_address=worker.address,
            to_address=req.to,
            amount=f"{req.amount} CLAW",
            status="success",
        )


@app.get("/gateway/workers/{worker_id}/balance", response_model=BalanceResponse)
def worker_balance(worker_id: str):
    """Get the CLAW balance for a worker's wallet."""
    with SessionFactory() as db:
        worker = db.query(Worker).filter(Worker.id == worker_id).first()
        if not worker:
            raise HTTPException(status_code=404, detail="Worker not found")

    bal = chain.query_balance(worker.address)
    return BalanceResponse(
        address=worker.address,
        balance_claw=f"{bal['claw']} CLAW",
        balance_aclaw=bal["aclaw"],
    )


@app.get("/gateway/workers", response_model=list[WorkerListItem])
def list_workers():
    """List all registered workers."""
    with SessionFactory() as db:
        workers = db.query(Worker).order_by(Worker.created_at.desc()).all()
        return [
            WorkerListItem(
                worker_id=w.id,
                name=w.name,
                address=w.address,
                active=w.active,
                heartbeats_sent=w.heartbeats_sent,
            )
            for w in workers
        ]


@app.get("/gateway/stats", response_model=GatewayStatsResponse)
def gateway_stats():
    """Get aggregate gateway statistics."""
    with SessionFactory() as db:
        total = db.query(Worker).count()
        active = db.query(Worker).filter(Worker.active == True).count()

        # Sum heartbeats sent today
        today_start = datetime.datetime.utcnow().replace(hour=0, minute=0, second=0, microsecond=0)
        # For simplicity, use total heartbeats_sent (resets require epoch tracking)
        total_heartbeats = sum(
            w.heartbeats_sent
            for w in db.query(Worker).filter(Worker.active == True).all()
        )

        balance = chain.get_operational_balance()

        return GatewayStatsResponse(
            total_registered=total,
            total_active=active,
            total_heartbeats_today=total_heartbeats,
            gateway_operational_balance=f"{balance} CLAW",
        )


@app.get("/gateway/health")
def health():
    """Health check endpoint."""
    return {"status": "ok", "service": "clawchain-gateway"}


# --- A2A Agent Card (Google Agent-to-Agent Protocol) ---

@app.get("/.well-known/agent.json")
def agent_card():
    """A2A Agent Card — describes this gateway's capabilities for agent discovery."""
    base_url = os.getenv("GATEWAY_BASE_URL", "https://api.clawchain.vsa.co.za")
    return JSONResponse(content={
        "name": "ClawChain Gateway",
        "description": (
            "ClawChain is a Cosmos SDK blockchain for AI agents. "
            "Workers earn CLAW tokens by sending heartbeat pings — no stake, "
            "no infrastructure, no binary required. Register via the Gateway API "
            "and start earning immediately."
        ),
        "url": base_url,
        "version": "1.0.0",
        "protocol": "a2a/1.0",
        "capabilities": {
            "streaming": False,
            "pushNotifications": False,
        },
        "skills": [
            {
                "id": "register-worker",
                "name": "Register as ClawChain Worker",
                "description": (
                    "Create a CLAW wallet and register as a worker on ClawChain. "
                    "Returns worker_id, wallet address, mnemonic, and ping token."
                ),
                "inputModes": ["application/json"],
                "outputModes": ["application/json"],
                "endpoint": f"{base_url}/gateway/workers/register",
            },
            {
                "id": "ping",
                "name": "Worker Heartbeat Ping",
                "description": (
                    "Signal liveness every 5 minutes to earn CLAW tokens. "
                    "The gateway proxies the heartbeat on-chain."
                ),
                "inputModes": ["application/json"],
                "outputModes": ["application/json"],
                "endpoint": f"{base_url}/gateway/workers/{'{worker_id}'}/ping",
            },
            {
                "id": "worker-status",
                "name": "Check Worker Status & Earnings",
                "description": "Query active status, heartbeat count, and earned CLAW.",
                "inputModes": ["application/json"],
                "outputModes": ["application/json"],
                "endpoint": f"{base_url}/gateway/workers/{'{worker_id}'}/status",
            },
            {
                "id": "send",
                "name": "Send CLAW Tokens",
                "description": (
                    "Transfer CLAW from your worker wallet to any claw1... address. "
                    "Requires X-Ping-Token header for authentication."
                ),
                "inputModes": ["application/json"],
                "outputModes": ["application/json"],
                "endpoint": f"{base_url}/gateway/workers/{'{worker_id}'}/send",
            },
            {
                "id": "balance",
                "name": "Check Wallet Balance",
                "description": "Query the CLAW balance of a worker's wallet.",
                "inputModes": ["application/json"],
                "outputModes": ["application/json"],
                "endpoint": f"{base_url}/gateway/workers/{'{worker_id}'}/balance",
            },
            {
                "id": "gateway-stats",
                "name": "Gateway Statistics",
                "description": "View total registered workers, active count, and heartbeats.",
                "inputModes": ["application/json"],
                "outputModes": ["application/json"],
                "endpoint": f"{base_url}/gateway/stats",
            },
        ],
        "provider": {
            "organization": "ClawChain",
            "url": "https://github.com/clawbotblockchain/clawchain",
        },
        "integrations": {
            "mcp": {
                "name": "ClawChain MCP Server",
                "description": "Connect Claude Code or Claude Desktop to ClawChain",
                "install": "npx clawchain-mcp",
                "source": "https://github.com/clawbotblockchain/clawchain/tree/main/mcp-server",
            },
            "openapi": f"{base_url}/openapi.json",
        },
        "links": {
            "explorer": "https://clawchain.vsa.co.za",
            "api_docs": f"{base_url}/docs",
            "documentation": "https://github.com/clawbotblockchain/clawchain/blob/main/skills/clawchain-worker/SKILL.md",
            "repository": "https://github.com/clawbotblockchain/clawchain",
        },
    })
