"""Background scheduler that proxies heartbeats for active workers."""

import asyncio
import datetime
import logging

from sqlalchemy.orm import Session

from . import chain
from .models import Worker

logger = logging.getLogger(__name__)

# Workers must ping within this window to receive heartbeat proxying
PING_TIMEOUT_MINUTES = 10
# How often the scheduler runs (slightly under the 5-min heartbeat interval)
SCHEDULER_INTERVAL_SECONDS = 240  # 4 minutes
# If every worker fails for this many consecutive ticks, emit a loud divergence alert.
# 3 ticks ≈ 12 minutes — long enough to rule out a single-block hiccup, short enough
# to catch the class of bug that silently burned 12 days of heartbeats.
DIVERGENCE_TICK_THRESHOLD = 3

# Module-level counter of consecutive ticks where every worker tx failed on-chain.
# Persists across ticks but resets to 0 the moment any heartbeat succeeds.
_all_failed_streak = 0


async def heartbeat_loop(session_factory):
    """Main scheduler loop — runs forever, proxying heartbeats for active workers."""
    logger.info("Heartbeat scheduler started (interval=%ds)", SCHEDULER_INTERVAL_SECONDS)
    while True:
        try:
            await asyncio.to_thread(_process_heartbeats, session_factory)
        except Exception:
            logger.exception("Error in heartbeat scheduler")
        await asyncio.sleep(SCHEDULER_INTERVAL_SECONDS)


def _process_heartbeats(session_factory):
    """Check all workers and send heartbeats for those with recent pings."""
    global _all_failed_streak
    cutoff = datetime.datetime.utcnow() - datetime.timedelta(minutes=PING_TIMEOUT_MINUTES)

    with session_factory() as db:
        # Find workers who have pinged recently
        active_workers = (
            db.query(Worker)
            .filter(Worker.active == True, Worker.last_ping != None, Worker.last_ping >= cutoff)
            .all()
        )

        if not active_workers:
            logger.debug("No active workers with recent pings")
            return

        logger.info("Processing heartbeats for %d active workers", len(active_workers))

        success_count = 0
        for worker in active_workers:
            try:
                tx_hash = chain.send_heartbeat(worker.key_name)
                if tx_hash:
                    worker.heartbeats_sent += 1
                    success_count += 1
                    logger.info(
                        "Heartbeat committed on-chain for %s (%s): tx=%s, total=%d",
                        worker.name, worker.address[:16], tx_hash[:16], worker.heartbeats_sent,
                    )
                else:
                    logger.warning("Heartbeat failed for %s (%s) — see preceding error for reason",
                                   worker.name, worker.address[:16])
            except Exception:
                logger.exception("Error sending heartbeat for %s", worker.name)

        # Divergence detector: if every worker tx failed for N consecutive ticks, the
        # gateway's view of "active" has drifted from the chain. Emit a loud, grep-able
        # alert so the next silent multi-day outage gets caught in minutes, not weeks.
        if success_count == 0 and active_workers:
            _all_failed_streak += 1
            if _all_failed_streak >= DIVERGENCE_TICK_THRESHOLD:
                logger.error(
                    "GATEWAY_CHAIN_DIVERGENCE: %d consecutive ticks with 0/%d heartbeats committed on-chain. "
                    "Gateway thinks these workers are active; chain is rejecting every tx. "
                    "Check preceding on-chain 'reason=' logs.",
                    _all_failed_streak, len(active_workers),
                )
        else:
            _all_failed_streak = 0

        # Deactivate workers who haven't pinged in time
        stale_workers = (
            db.query(Worker)
            .filter(Worker.active == True, Worker.last_ping != None, Worker.last_ping < cutoff)
            .all()
        )
        for worker in stale_workers:
            worker.active = False
            logger.info("Deactivated stale worker: %s (%s), last ping: %s",
                        worker.name, worker.address[:16], worker.last_ping)

        db.commit()
