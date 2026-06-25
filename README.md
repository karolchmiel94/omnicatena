# Omnicatena

A single-tenant, multi-chain wallet & transaction platform. Create and manage
wallets, check balances, and transfer native value across **Bitcoin, Ethereum,
Base, Solana, and TRON** through one consistent interface — exposed over both an
**HTTP API** and a **CLI**.

## Why this exists

The interesting engineering problem here is presenting five very different
blockchains behind **common `Wallet` and `Transaction` abstractions** without
letting their differences leak into the core. The architecture is
[ports & adapters (hexagonal)](./ARCHITECTURE.md): the domain and use cases know
nothing about any specific chain, RPC client, datastore, or message broker —
those are all swappable adapters.

## Documents

| Doc | Purpose |
|-----|---------|
| [REQUIREMENTS.md](./REQUIREMENTS.md) | Functional & non-functional requirements, non-goals |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Hexagonal design, ports, directory layout, the unsigned-tx envelope |
| [ROADMAP.md](./ROADMAP.md) | One-week V1 plan, then V2 (interoperability, traffic/cost monitoring) |
| [adr/](./adr/) | Architecture Decision Records — *why* each choice was made |
| [turba](https://github.com/karolchmiel94/turba) | Traffic simulator — noise generator + user simulator, congestion profiles (separate repo) |

## Quick start (once Go is installed)

```bash
make up                      # start local chain nodes + Kafka via Docker
make run-api                 # start the HTTP API (EVM + Bitcoin working; Solana/Base/TRON Day 4-5)
make run-cli -- wallet create --label demo
```

## Supported chains

| Chain | Family | V1 network | Notes |
|-------|--------|-----------|-------|
| Bitcoin | UTXO | local `regtest` | ✓ working — mine blocks to self-fund |
| Ethereum | EVM | local (Anvil) | ✓ working — pre-funded dev accounts |
| Base | EVM | local (Anvil, alt chainId) | shares the EVM adapter — Day 5 |
| Solana | Solana | local (`solana-test-validator`) | ed25519 keys — Day 4 |
| TRON | TVM | local (`tron-quickstart`) | heaviest local node — Day 5 |
