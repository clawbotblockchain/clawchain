"""Chain interaction layer — wraps clawchaind CLI for signing and broadcasting."""

import json
import logging
import os
import subprocess
import time

logger = logging.getLogger(__name__)

BINARY = os.getenv("CLAWCHAIND_PATH", "clawchaind")
CHAIN_ID = os.getenv("CHAIN_ID", "clawchain-1")
NODE_URL = os.getenv("NODE_URL", "tcp://localhost:26657")
KEYRING_BACKEND = "test"
GATEWAY_HOME = os.getenv("GATEWAY_HOME", os.path.expanduser("~/.clawchain-gateway"))
GATEWAY_KEY = os.getenv("GATEWAY_KEY_NAME", "gateway-operational")
FUND_AMOUNT = os.getenv("FUND_AMOUNT", "1aclaw")  # Minimal funding for account creation


def _run(args: list[str], timeout: int = 30) -> subprocess.CompletedProcess:
    """Run a clawchaind command and return the result."""
    result = subprocess.run(args, capture_output=True, text=True, timeout=timeout)
    if result.returncode != 0:
        logger.error("Command failed: %s\nstderr: %s", " ".join(args[:5]), result.stderr)
    return result


def _common_tx_flags() -> list[str]:
    return [
        "--chain-id", CHAIN_ID,
        "--node", NODE_URL,
        "--keyring-backend", KEYRING_BACKEND,
        "--home", GATEWAY_HOME,
        "--fees", "0aclaw",
        "--yes",
        "--output", "json",
    ]


def init_gateway_home():
    """Initialize the gateway home directory if it doesn't exist."""
    os.makedirs(GATEWAY_HOME, exist_ok=True)
    # Check if gateway operational key exists
    result = _run([
        BINARY, "keys", "show", GATEWAY_KEY,
        "--keyring-backend", KEYRING_BACKEND,
        "--home", GATEWAY_HOME,
    ])
    if result.returncode != 0:
        logger.warning(
            "Gateway operational key '%s' not found. "
            "Create it with: %s keys add %s --keyring-backend test --home %s",
            GATEWAY_KEY, BINARY, GATEWAY_KEY, GATEWAY_HOME,
        )


def create_worker_key(worker_id: str) -> dict:
    """Create a new Cosmos keypair for a worker. Returns address and mnemonic."""
    key_name = f"worker-{worker_id}"
    result = _run([
        BINARY, "keys", "add", key_name,
        "--keyring-backend", KEYRING_BACKEND,
        "--home", GATEWAY_HOME,
        "--output", "json",
    ])
    if result.returncode != 0:
        raise RuntimeError(f"Failed to create key: {result.stderr}")

    key_info = json.loads(result.stdout)
    address = key_info.get("address", "")

    # Extract mnemonic from stderr (Cosmos SDK outputs it there)
    mnemonic = ""
    lines = result.stderr.strip().split("\n")
    for line in lines:
        words = line.strip().split()
        if len(words) >= 12 and all(w.isalpha() for w in words[:12]):
            mnemonic = line.strip()
            break

    return {
        "key_name": key_name,
        "address": address,
        "mnemonic": mnemonic,
    }


def fund_account(to_address: str) -> str | None:
    """Send minimal CLAW from operational key to create the worker account on-chain."""
    result = _run([
        BINARY, "tx", "bank", "send",
        GATEWAY_KEY, to_address, FUND_AMOUNT,
        *_common_tx_flags(),
    ])
    if result.returncode != 0:
        logger.error("Failed to fund %s: %s", to_address, result.stderr)
        return None
    try:
        tx_data = json.loads(result.stdout)
        return tx_data.get("txhash")
    except json.JSONDecodeError:
        return None


def register_worker(key_name: str, worker_name: str) -> str | None:
    """Register a worker on-chain."""
    result = _run([
        BINARY, "tx", "participation", "register-worker",
        "--name", worker_name,
        "--from", key_name,
        *_common_tx_flags(),
    ])
    if result.returncode != 0:
        logger.error("Failed to register worker %s: %s", key_name, result.stderr)
        return None
    try:
        tx_data = json.loads(result.stdout)
        return tx_data.get("txhash")
    except json.JSONDecodeError:
        return None


def _wait_for_tx_commit(txhash: str, timeout_seconds: int = 10) -> tuple[int, str]:
    """Poll `clawchaind query tx` until the tx is indexed or timeout elapses.

    Returns (code, raw_log). code=0 means committed successfully. code>0 means the tx
    landed in a block but the handler rejected it — raw_log carries the chain's reason.
    A synthetic code=-1 means the tx never indexed within the timeout (broadcast failed
    or tx is still pending — treat as failure for gateway purposes).
    """
    deadline = time.monotonic() + timeout_seconds
    while time.monotonic() < deadline:
        result = _run([
            BINARY, "query", "tx", txhash,
            "--node", NODE_URL,
            "--output", "json",
        ], timeout=5)
        if result.returncode == 0 and result.stdout.strip():
            try:
                data = json.loads(result.stdout)
                return int(data.get("code", 0)), str(data.get("raw_log", ""))
            except json.JSONDecodeError:
                pass
        time.sleep(1)
    return -1, "tx not indexed within timeout"


def send_heartbeat(key_name: str) -> str | None:
    """Send a heartbeat transaction for a worker.

    Returns the txhash ONLY if the tx was committed on-chain with code 0.
    Returns None (and logs the on-chain error) if the tx failed at any stage:
    broadcast rejected, tx timed out without indexing, or chain rejected the handler.
    """
    result = _run([
        BINARY, "tx", "participation", "worker-heartbeat",
        "--from", key_name,
        *_common_tx_flags(),
    ])
    if result.returncode != 0:
        logger.error("Heartbeat broadcast failed for %s: %s", key_name, result.stderr.strip())
        return None
    try:
        tx_data = json.loads(result.stdout)
    except json.JSONDecodeError:
        logger.error("Heartbeat broadcast returned non-JSON output for %s: %s", key_name, result.stdout[:200])
        return None

    broadcast_code = int(tx_data.get("code", 0))
    txhash = tx_data.get("txhash")
    if broadcast_code != 0:
        logger.error("Heartbeat broadcast rejected for %s: code=%d raw_log=%s",
                     key_name, broadcast_code, tx_data.get("raw_log", ""))
        return None
    if not txhash:
        logger.error("Heartbeat broadcast returned no txhash for %s", key_name)
        return None

    commit_code, raw_log = _wait_for_tx_commit(txhash)
    if commit_code != 0:
        logger.error("Heartbeat rejected on-chain for %s: tx=%s code=%d reason=%s",
                     key_name, txhash, commit_code, raw_log)
        return None
    return txhash


def send_tokens(from_key: str, to_address: str, amount_aclaw: str) -> dict:
    """Send CLAW from a worker's key to another address. Returns {txhash, code} or raises."""
    result = _run([
        BINARY, "tx", "bank", "send",
        from_key, to_address, amount_aclaw,
        *_common_tx_flags(),
    ])
    if result.returncode != 0:
        logger.error("Send failed from %s to %s: %s", from_key, to_address, result.stderr)
        raise RuntimeError(result.stderr.strip() or "Transaction failed")
    try:
        tx_data = json.loads(result.stdout)
        return {"txhash": tx_data.get("txhash", ""), "code": tx_data.get("code", 0)}
    except json.JSONDecodeError:
        raise RuntimeError("Could not parse transaction result")


def query_balance(address: str) -> dict:
    """Query balance for an address. Returns {aclaw: str, claw: str}."""
    result = _run([
        BINARY, "query", "bank", "balances", address,
        "--home", GATEWAY_HOME,
        "--node", NODE_URL,
        "--output", "json",
    ])
    if result.returncode != 0:
        return {"aclaw": "0", "claw": "0"}
    try:
        data = json.loads(result.stdout)
        for coin in data.get("balances", []):
            if coin["denom"] == "aclaw":
                aclaw = coin["amount"]
                claw = int(aclaw) / (10**18)
                return {"aclaw": aclaw, "claw": f"{claw:,.6f}"}
        return {"aclaw": "0", "claw": "0"}
    except (json.JSONDecodeError, KeyError):
        return {"aclaw": "0", "claw": "0"}


def query_worker(address: str) -> dict | None:
    """Query worker info from the chain."""
    result = _run([
        BINARY, "query", "participation", "get-worker-info",
        "--address", address,
        "--home", GATEWAY_HOME,
        "--node", NODE_URL,
        "--output", "json",
    ])
    if result.returncode != 0:
        return None
    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError:
        return None


def query_worker_rewards(address: str) -> dict | None:
    """Query worker rewards from the chain."""
    result = _run([
        BINARY, "query", "participation", "worker-rewards",
        "--address", address,
        "--home", GATEWAY_HOME,
        "--node", NODE_URL,
        "--output", "json",
    ])
    if result.returncode != 0:
        return None
    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError:
        return None


def query_worker_stats() -> dict | None:
    """Query aggregate worker stats from the chain."""
    result = _run([
        BINARY, "query", "participation", "worker-stats",
        "--home", GATEWAY_HOME,
        "--node", NODE_URL,
        "--output", "json",
    ])
    if result.returncode != 0:
        return None
    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError:
        return None


def get_operational_balance() -> str:
    """Get the balance of the gateway operational key."""
    result = _run([
        BINARY, "keys", "show", GATEWAY_KEY, "-a",
        "--keyring-backend", KEYRING_BACKEND,
        "--home", GATEWAY_HOME,
    ])
    if result.returncode != 0:
        return "0"
    address = result.stdout.strip()
    bal_result = _run([
        BINARY, "query", "bank", "balances", address,
        "--home", GATEWAY_HOME,
        "--node", NODE_URL,
        "--output", "json",
    ])
    if bal_result.returncode != 0:
        return "0"
    try:
        data = json.loads(bal_result.stdout)
        for coin in data.get("balances", []):
            if coin["denom"] == "aclaw":
                # Convert aclaw to CLAW (divide by 10^18)
                aclaw = int(coin["amount"])
                claw = aclaw // (10**18)
                return f"{claw:,}"
        return "0"
    except (json.JSONDecodeError, KeyError):
        return "0"
