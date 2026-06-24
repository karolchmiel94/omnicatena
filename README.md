# Omnicatena

A single-tenant, multi-chain wallet & transaction platform. Create and manage
wallets, check balances, and transfer native value across **Bitcoin, Ethereum,
Base, Solana, and TRON** through one consistent interface — exposed over both an
**HTTP API** and a **CLI**.

> Status: greenfield prototype (V1). Built as a learning/portfolio project, run
> locally. Not production-hardened.

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

## Quick start (once Go is installed)

```bash
sudo pacman -S go            # Go is not yet installed on this machine
make tidy                    # resolve module dependencies
make up                      # start local chain nodes + Kafka via Docker
make run-api                 # start the HTTP API
make run-cli -- wallet create --label demo
```

## Supported chains

| Chain | Family | V1 network | Notes |
|-------|--------|-----------|-------|
| Bitcoin | UTXO | local `regtest` | mine blocks to self-fund |
| Ethereum | EVM | local (Anvil) | pre-funded dev accounts |
| Base | EVM | local (Anvil, alt chainId) | shares the EVM adapter with Ethereum |
| Solana | Solana | local (`solana-test-validator`) | ed25519 keys |
| TRON | TVM | local (`tron-quickstart`) | heaviest local node |

> The module path `github.com/karolchmiel94/omnicatena` is a placeholder — rename in
> `go.mod` to match your actual repo.
