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

        for worker in active_workers:
            try:
                tx_hash = chain.send_heartbeat(worker.key_name)
                if tx_hash:
                    worker.heartbeats_sent += 1
                    logger.info(
                        "Heartbeat sent for %s (%s): tx=%s, total=%d",
                        worker.name, worker.address[:16], tx_hash[:16], worker.heartbeats_sent,
                    )
                else:
                    logger.warning("Heartbeat failed for %s (%s)", worker.name, worker.address[:16])
            except Exception:
                logger.exception("Error sending heartbeat for %s", worker.name)

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
