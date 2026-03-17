#!/bin/bash
set -euo pipefail

# ClawChain Mainnet Initialization
# Creates 3 seed validator nodes on localhost
# Chain ID: clawchain-1 (PRODUCTION)

CHAIN_ID="clawchain-1"
DENOM="aclaw"
BINARY="clawchaind"
HOME_BASE="${HOME}/.clawchain-localnet"
# Production epoch: 24 hours (86400 seconds)
EPOCH_DURATION=86400
# ~90B CLAW total (6.75B founder + 74.75B pool + 10B treasury + 600K validators)
TOTAL_SUPPLY="91500600000000000000000000000"
# Founder: 6.75B CLAW (2-year linear vest, no cliff)
FOUNDER_AMOUNT="6750000000000000000000000000"
# Reward pool: 74.75B CLAW (~5.46 years at 37.5M/day)
REWARD_POOL_AMOUNT="74750000000000000000000000000"
# Treasury: 10B CLAW
TREASURY_AMOUNT="10000000000000000000000000000"
# Founder vesting: 2 years = 730 days = 63072000 seconds
FOUNDER_VEST_DURATION=63072000
# Each genesis validator gets 100K CLAW for staking
VAL_STAKE="100000000000000000000000"
# Daily reward amount (~37.5M CLAW in aclaw)
DAILY_REWARD="37500000000000000000000000"
# Min stake: 100K CLAW
MIN_STAKE="100000000000000000000000"

NUM_VALIDATORS=3
BASE_P2P_PORT=26656
BASE_RPC_PORT=26657
BASE_API_PORT=1317
BASE_GRPC_PORT=9090

echo "=========================================="
echo "   ClawChain Mainnet Init"
echo "   Chain ID: $CHAIN_ID"
echo "   THIS IS A PRODUCTION GENESIS"
echo "=========================================="
echo ""
echo "Validators: $NUM_VALIDATORS"
echo "Epoch: ${EPOCH_DURATION}s (24 hours)"
echo ""

# Clean previous state
rm -rf "$HOME_BASE"

# Generate validator keys and init nodes
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    NODE_HOME="$HOME_BASE/node$i"
    MONIKER="validator-$i"

    echo "--- Initializing node $i ($MONIKER) ---"
    $BINARY init "$MONIKER" --chain-id "$CHAIN_ID" --home "$NODE_HOME" > /dev/null 2>&1

    # Create validator key
    $BINARY keys add "val$i" --keyring-backend test --home "$NODE_HOME" > /dev/null 2>&1

    # Create founder, treasury keys on node 0 only — SAVE MNEMONICS
    if [ $i -eq 0 ]; then
        echo ""
        echo "=========================================="
        echo "  CREATING FOUNDER KEY — SAVE THIS MNEMONIC!"
        echo "=========================================="
        $BINARY keys add founder --keyring-backend test --home "$NODE_HOME" 2>&1 | tee "$HOME_BASE/founder-key-backup.txt"
        echo ""
        echo "=========================================="
        echo "  CREATING TREASURY KEY — SAVE THIS MNEMONIC!"
        echo "=========================================="
        $BINARY keys add treasury --keyring-backend test --home "$NODE_HOME" 2>&1 | tee "$HOME_BASE/treasury-key-backup.txt"
        echo ""
        echo "  Key backups saved to:"
        echo "    $HOME_BASE/founder-key-backup.txt"
        echo "    $HOME_BASE/treasury-key-backup.txt"
        echo "  *** SECURE THESE FILES AND DELETE FROM SERVER ***"
        echo ""
    fi
done

# Use node0's genesis as the master
MASTER_HOME="$HOME_BASE/node0"
GENESIS="$MASTER_HOME/config/genesis.json"

# Get addresses
FOUNDER_ADDR=$($BINARY keys show founder -a --keyring-backend test --home "$MASTER_HOME")
TREASURY_ADDR=$($BINARY keys show treasury -a --keyring-backend test --home "$MASTER_HOME")

echo ""
echo "Founder address: $FOUNDER_ADDR"
echo "Treasury address: $TREASURY_ADDR"

# Configure genesis - update denom from default "stake" to "aclaw"
sed -i "s/\"stake\"/\"$DENOM\"/g" "$GENESIS"

# Add genesis accounts
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    NODE_HOME="$HOME_BASE/node$i"
    VAL_ADDR=$($BINARY keys show "val$i" -a --keyring-backend test --home "$NODE_HOME")
    echo "Validator $i address: $VAL_ADDR"

    # Each validator gets enough to stake + gas
    VAL_TOTAL="200000000000000000000000" # 200K CLAW
    $BINARY genesis add-genesis-account "$VAL_ADDR" "${VAL_TOTAL}${DENOM}" --home "$MASTER_HOME" --keyring-backend test
done

# Add founder account (6.75B CLAW with 2-year linear vest)
VEST_START=$(date +%s)
VEST_END=$((VEST_START + FOUNDER_VEST_DURATION))
$BINARY genesis add-genesis-account "$FOUNDER_ADDR" "${FOUNDER_AMOUNT}${DENOM}" \
    --vesting-amount "${FOUNDER_AMOUNT}${DENOM}" \
    --vesting-start-time "$VEST_START" \
    --vesting-end-time "$VEST_END" \
    --home "$MASTER_HOME" --keyring-backend test
echo "Founder vesting: start=$(date -d @$VEST_START +%Y-%m-%d) end=$(date -d @$VEST_END +%Y-%m-%d) (2 years linear)"

# Add treasury account (10B CLAW)
$BINARY genesis add-genesis-account "$TREASURY_ADDR" "${TREASURY_AMOUNT}${DENOM}" --home "$MASTER_HOME" --keyring-backend test

# Fund reward pool module account (50B CLAW) and update params
# Module accounts can't be added via CLI — inject directly into genesis JSON
python3 - "$GENESIS" "$EPOCH_DURATION" "$MIN_STAKE" "$DAILY_REWARD" "$REWARD_POOL_AMOUNT" "$DENOM" <<'PYEOF'
import json, sys, hashlib

genesis_file = sys.argv[1]
epoch_duration = int(sys.argv[2])
min_stake = sys.argv[3]
daily_reward = sys.argv[4]
reward_pool_amount = sys.argv[5]
denom = sys.argv[6]

with open(genesis_file, 'r') as f:
    genesis = json.load(f)

# --- Compute module account address ---
# Cosmos SDK: address = sha256(moduleName)[:20], then bech32("claw", addr_bytes)
module_name = "participation_reward_pool"
addr_bytes = hashlib.sha256(module_name.encode()).digest()[:20]

# Bech32 encoding
CHARSET = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
def bech32_polymod(values):
    GEN = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3]
    chk = 1
    for v in values:
        b = (chk >> 25)
        chk = (chk & 0x1ffffff) << 5 ^ v
        for i in range(5):
            chk ^= GEN[i] if ((b >> i) & 1) else 0
    return chk

def bech32_hrp_expand(hrp):
    return [ord(x) >> 5 for x in hrp] + [0] + [ord(x) & 31 for x in hrp]

def bech32_create_checksum(hrp, data):
    values = bech32_hrp_expand(hrp) + data
    polymod = bech32_polymod(values + [0, 0, 0, 0, 0, 0]) ^ 1
    return [(polymod >> 5 * (5 - i)) & 31 for i in range(6)]

def convertbits(data, frombits, tobits, pad=True):
    acc = 0
    bits = 0
    ret = []
    maxv = (1 << tobits) - 1
    for value in data:
        acc = (acc << frombits) | value
        bits += frombits
        while bits >= tobits:
            bits -= tobits
            ret.append((acc >> bits) & maxv)
    if pad:
        if bits:
            ret.append((acc << (tobits - bits)) & maxv)
    return ret

hrp = "claw"
data5bit = convertbits(list(addr_bytes), 8, 5)
checksum = bech32_create_checksum(hrp, data5bit)
module_addr = hrp + "1" + "".join([CHARSET[d] for d in data5bit + checksum])
print(f"Reward pool module address: {module_addr}")

# --- Add module account to auth.accounts ---
module_account_entry = {
    "@type": "/cosmos.auth.v1beta1.ModuleAccount",
    "base_account": {
        "address": module_addr,
        "pub_key": None,
        "account_number": "0",
        "sequence": "0"
    },
    "name": module_name,
    "permissions": ["burner"]
}
genesis['app_state']['auth']['accounts'].append(module_account_entry)

# --- Add balance for module account ---
balance_entry = {
    "address": module_addr,
    "coins": [{"denom": denom, "amount": reward_pool_amount}]
}
genesis['app_state']['bank']['balances'].append(balance_entry)

# --- Update total supply ---
# Find existing supply entry for aclaw and add reward pool amount
supply = genesis['app_state']['bank']['supply']
found = False
for coin in supply:
    if coin['denom'] == denom:
        old_amount = int(coin['amount'])
        coin['amount'] = str(old_amount + int(reward_pool_amount))
        found = True
        print(f"Updated supply: {old_amount} -> {coin['amount']}")
        break
if not found:
    supply.append({"denom": denom, "amount": reward_pool_amount})
    print(f"Added supply: {reward_pool_amount}")

# --- Sort balances by address (Cosmos SDK requirement) ---
genesis['app_state']['bank']['balances'].sort(key=lambda x: x['address'])

# --- Update staking params ---
genesis['app_state']['staking']['params']['bond_denom'] = 'aclaw'
genesis['app_state']['staking']['params']['unbonding_time'] = '604800s'  # 7 days
genesis['app_state']['staking']['params']['max_validators'] = 125

# --- Update governance params ---
genesis['app_state']['gov']['params']['min_deposit'] = [{'denom': 'aclaw', 'amount': '10000000000000000000000'}]  # 10K CLAW
genesis['app_state']['gov']['params']['expedited_min_deposit'] = [{'denom': 'aclaw', 'amount': '50000000000000000000000'}]  # 50K CLAW

# --- Update mint params - disable inflation ---
genesis['app_state']['mint']['minter']['inflation'] = '0.000000000000000000'
genesis['app_state']['mint']['params']['inflation_rate_change'] = '0.000000000000000000'
genesis['app_state']['mint']['params']['inflation_max'] = '0.000000000000000000'
genesis['app_state']['mint']['params']['inflation_min'] = '0.000000000000000000'
genesis['app_state']['mint']['params']['mint_denom'] = 'aclaw'

# --- Update participation params ---
if 'participation' in genesis['app_state']:
    genesis['app_state']['participation']['params']['epoch_duration'] = str(epoch_duration)
    genesis['app_state']['participation']['params']['min_stake'] = min_stake
    genesis['app_state']['participation']['params']['stake_weight'] = '20'
    genesis['app_state']['participation']['params']['activity_weight'] = '60'
    genesis['app_state']['participation']['params']['uptime_weight'] = '20'
    genesis['app_state']['participation']['params']['daily_reward_amount'] = daily_reward
    genesis['app_state']['participation']['params']['worker_reward_ratio'] = '60'
    genesis['app_state']['participation']['params']['heartbeat_interval'] = '300'
    genesis['app_state']['participation']['params']['max_missed_heartbeats'] = '100'
    genesis['app_state']['participation']['params']['max_workers'] = '0'  # 0 = unlimited

# --- Update slashing params ---
genesis['app_state']['slashing']['params']['signed_blocks_window'] = '1000'
genesis['app_state']['slashing']['params']['min_signed_per_window'] = '0.500000000000000000'
genesis['app_state']['slashing']['params']['downtime_jail_duration'] = '600s'
genesis['app_state']['slashing']['params']['slash_fraction_double_sign'] = '0.200000000000000000'
genesis['app_state']['slashing']['params']['slash_fraction_downtime'] = '0.050000000000000000'

# --- Update consensus params for 3s block time ---
if 'consensus' in genesis and 'params' in genesis['consensus']:
    genesis['consensus']['params']['block']['max_bytes'] = '22020096'
    genesis['consensus']['params']['block']['max_gas'] = '100000000'

with open(genesis_file, 'w') as f:
    json.dump(genesis, f, indent=2)

print(f"Genesis updated: chain=clawchain-1, epoch={epoch_duration}s (24h), reward_pool={reward_pool_amount}")
PYEOF

# Create gentx for each validator
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    NODE_HOME="$HOME_BASE/node$i"

    if [ $i -ne 0 ]; then
        # Copy master genesis to other nodes
        cp "$GENESIS" "$NODE_HOME/config/genesis.json"
    fi

    $BINARY genesis gentx "val$i" "${VAL_STAKE}${DENOM}" \
        --chain-id "$CHAIN_ID" \
        --moniker "validator-$i" \
        --keyring-backend test \
        --home "$NODE_HOME" > /dev/null 2>&1

    echo "Created gentx for validator-$i"
done

# Collect gentxs on node0
for i in $(seq 1 $((NUM_VALIDATORS - 1))); do
    cp "$HOME_BASE/node$i/config/gentx/"* "$MASTER_HOME/config/gentx/"
done

$BINARY genesis collect-gentxs --home "$MASTER_HOME" > /dev/null 2>&1
echo "Collected all gentxs"

# Validate genesis
$BINARY genesis validate-genesis --home "$MASTER_HOME"
echo "Genesis validated successfully"

# Distribute final genesis and configure each node
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    NODE_HOME="$HOME_BASE/node$i"

    # Copy final genesis
    if [ $i -ne 0 ]; then
        cp "$GENESIS" "$NODE_HOME/config/genesis.json"
    fi

    # Configure ports to avoid conflicts
    P2P_PORT=$((BASE_P2P_PORT + i * 100))
    RPC_PORT=$((BASE_RPC_PORT + i * 100))
    API_PORT=$((BASE_API_PORT + i * 100))
    GRPC_PORT=$((BASE_GRPC_PORT + i * 100))
    PPROF_PORT=$((6060 + i))

    # Update config.toml
    CONFIG="$NODE_HOME/config/config.toml"
    sed -i "s/laddr = \"tcp:\/\/0.0.0.0:26656\"/laddr = \"tcp:\/\/0.0.0.0:$P2P_PORT\"/" "$CONFIG"
    sed -i "s/laddr = \"tcp:\/\/127.0.0.1:26657\"/laddr = \"tcp:\/\/127.0.0.1:$RPC_PORT\"/" "$CONFIG"
    sed -i "s/pprof_laddr = \"localhost:6060\"/pprof_laddr = \"localhost:$PPROF_PORT\"/" "$CONFIG"
    # Set 3s block time
    sed -i 's/timeout_commit = ".*"/timeout_commit = "3s"/' "$CONFIG"
    # Allow duplicate IPs (all validators on localhost)
    sed -i 's/allow_duplicate_ip = false/allow_duplicate_ip = true/' "$CONFIG"

    # Update app.toml
    APP_TOML="$NODE_HOME/config/app.toml"
    sed -i "s/address = \"tcp:\/\/localhost:1317\"/address = \"tcp:\/\/localhost:$API_PORT\"/" "$APP_TOML"
    sed -i "s/address = \"localhost:9090\"/address = \"localhost:$GRPC_PORT\"/" "$APP_TOML"
    sed -i 's/minimum-gas-prices = ""/minimum-gas-prices = "0aclaw"/' "$APP_TOML"
    sed -i 's/enable = false/enable = true/' "$APP_TOML"  # Enable API

    echo "Configured node $i: P2P=$P2P_PORT RPC=$RPC_PORT API=$API_PORT"
done

# Build persistent_peers list
PEERS=""
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    NODE_HOME="$HOME_BASE/node$i"
    NODE_ID=$($BINARY tendermint show-node-id --home "$NODE_HOME" 2>/dev/null || $BINARY comet show-node-id --home "$NODE_HOME" 2>/dev/null)
    P2P_PORT=$((BASE_P2P_PORT + i * 100))
    if [ -n "$PEERS" ]; then
        PEERS="$PEERS,"
    fi
    PEERS="${PEERS}${NODE_ID}@127.0.0.1:${P2P_PORT}"
done

# Set persistent_peers on all nodes
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    NODE_HOME="$HOME_BASE/node$i"
    CONFIG="$NODE_HOME/config/config.toml"
    sed -i "s|^persistent_peers = .*|persistent_peers = \"$PEERS\"|" "$CONFIG"
done

# Save genesis copy for distribution
cp "$GENESIS" "$HOME_BASE/genesis.json"
echo ""
echo "Production genesis saved to: $HOME_BASE/genesis.json"

echo ""
echo "=========================================="
echo "   ClawChain Mainnet Initialized"
echo "=========================================="
echo "Nodes: $HOME_BASE/node{0..$((NUM_VALIDATORS-1))}"
echo "Chain ID: $CHAIN_ID"
echo "Denom: $DENOM (1 CLAW = 10^18 $DENOM)"
echo "Epoch: ${EPOCH_DURATION}s (24 hours)"
echo ""
echo "Addresses:"
echo "  Founder: $FOUNDER_ADDR"
echo "  Treasury: $TREASURY_ADDR"
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    NODE_HOME="$HOME_BASE/node$i"
    VAL_ADDR=$($BINARY keys show "val$i" -a --keyring-backend test --home "$NODE_HOME")
    echo "  Validator $i: $VAL_ADDR"
done
echo ""
echo "IMPORTANT:"
echo "  1. Secure founder-key-backup.txt and treasury-key-backup.txt"
echo "  2. Run: bash scripts/localnet-start.sh"
echo "  3. Verify: curl -s http://127.0.0.1:26657/status"
