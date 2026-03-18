"""Seed the Gateway DB with the 3 existing ClawChain bot workers."""

import os
import sys

# Load .env manually
env_path = os.path.join(os.path.dirname(__file__), ".env")
if os.path.exists(env_path):
    with open(env_path) as f:
        for line in f:
            line = line.strip()
            if line and not line.startswith("#") and "=" in line:
                k, v = line.split("=", 1)
                os.environ.setdefault(k.strip(), v.strip())

from models import Worker, get_engine, get_session_factory, init_db

DATABASE_URL = os.getenv("DATABASE_URL", "sqlite:///./data/gateway.db")
os.makedirs("data", exist_ok=True)
engine = get_engine(DATABASE_URL)
init_db(engine)
SessionFactory = get_session_factory(engine)

BOTS = [
    {
        "id": "clawchainbot",
        "name": "ClawChainBot",
        "platform": "moltbook",
        "key_name": "clawchainbot",
        "address": "claw18ke09pswzxwj2axndx4hm72l44yp2843casq68",
    },
    {
        "id": "clawchaindev",
        "name": "ClawChainDev",
        "platform": "moltbook",
        "key_name": "clawchaindev",
        "address": "claw1qrl45zp9wkaxzpf3njcgnjpukqhv46pvwt0l3k",
    },
    {
        "id": "clawchainjoin",
        "name": "ClawChainJoin",
        "platform": "moltbook",
        "key_name": "clawchainjoin",
        "address": "claw1zffwd3qd52h76dn4vfsqm7nf3uxxhyd0j3fa06",
    },
]

with SessionFactory() as db:
    for bot in BOTS:
        existing = db.query(Worker).filter(Worker.id == bot["id"]).first()
        if existing:
            print(f"  Already exists: {bot['name']} ({bot['id']})")
            continue
        worker = Worker(
            id=bot["id"],
            name=bot["name"],
            platform=bot["platform"],
            key_name=bot["key_name"],
            address=bot["address"],
            active=True,
        )
        db.add(worker)
        print(f"  Seeded: {bot['name']} ({bot['id']}) -> {bot['address']}")
    db.commit()

print("Done. Bot workers seeded into Gateway DB.")
