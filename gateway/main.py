"""ClawChain Gateway — Zero-infrastructure worker heartbeat proxy."""

import asyncio
import datetime
import logging
import os

from contextlib import asynccontextmanager
from fastapi import FastAPI, Header, HTTPException
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
MIN_PING_INTERVAL_SECONDS = 240  # 4 minutes

# Database setup
DATABASE_URL = os.getenv("DATABASE_URL", "sqlite:///./data/gateway.db")
os.makedirs("data", exist_ok=True)
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
