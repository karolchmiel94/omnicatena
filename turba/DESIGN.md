# Turba

Network traffic simulator for Omnicatena. Provides controllable blockchain environments and
parametrizable load generation to test fee estimation, confirmation latency, and API throughput
under realistic conditions.

## Architecture

Two independent async layers run concurrently:

**Noise generator** — bypasses the Omnicatena API entirely. Talks directly to chain RPC nodes and
floods the mempool with background transactions from pre-funded dev accounts. This simulates other
network participants and is what actually drives gas prices up (EIP-1559 base fee responds to block
fullness) and increases confirmation wait times.

**User simulator** — hits the Omnicatena HTTP API with realistic wallet lifecycle operations:
1. `POST /wallets` — create wallet
2. Fund the new address from a chain dev account (via chain RPC, not the API)
3. `POST /transactions` — transfer to a second wallet
4. `GET /transactions/{hash}` — poll until confirmed

## Congestion profiles

Three named presets; all parameters individually overridable via CLI flags.

| Parameter           | `low` | `medium` | `high` |
|---------------------|-------|----------|--------|
| API users (concurrent) | 5  | 25       | 100    |
| Noise tx/s (chain-direct) | 0 | 15    | 60     |
| EVM block time      | instant | 2s     | 12s    |
| BTC block interval  | 10s   | 30s      | 120s   |

High profile = full blocks every slot, base fee spiking, confirmation queue backing up.
Low profile = quiet devnet, instant confirmations, baseline fee estimates.

## Metrics

Live summary printed every 30s, final report at exit:

```
[t=60s]  submitted: 847  confirmed: 791  failed: 3
         confirm p50: 14s   p95: 38s
         fee estimate p50: 12 gwei   p95: 31 gwei
         api latency p50: 8ms   p95: 22ms
```

Optional JSON report via `--out report.json`.

## File structure

```
turba/
  simulate.py     # CLI entrypoint — argument parsing, wires all components
  profiles.py     # preset definitions and CLI param merging
  noise.py        # chain-direct mempool spammer (web3.py / python-bitcoinrpc / solders)
  users.py        # async API user scenarios (httpx + asyncio)
  metrics.py      # counters, histograms, periodic reporter
```

## Chain support

| Chain       | Docker image                        | Notes                                      |
|-------------|-------------------------------------|--------------------------------------------|
| EVM (Eth/Base) | `ghcr.io/foundry-rs/foundry` (Anvil) | 10 accounts × 10k ETH, configurable block time |
| Bitcoin     | `bitcoin/bitcoin` (regtest)         | blocks mined on demand via RPC             |
| Solana      | `solanalabs/solana` (test-validator)| `solana airdrop` for funding               |
| TRON        | `trontools/quickstart`              | pre-funded test accounts                   |

## Usage

```bash
# quiet run against EVM for 5 minutes
python turba/simulate.py --profile low --chain evm --duration 300

# high congestion, custom overrides, JSON report
python turba/simulate.py --profile high --users 200 --noise 80 --out report.json
```

## Block time note

Anvil block time is set at startup via `--block-time N`. The profile documents the expected value
but does not restart containers — set it in docker-compose before running, or pass `--block-time`
explicitly to Anvil when launching.
