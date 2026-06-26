# Roadmap

## V1 — one week

The strategy is **walking skeleton first**: get one chain working end to end
through the whole hexagon before replicating across the other four. This
validates the architecture in code while it's still cheap to change.

### ✓ Day 1 — Foundations
- Project skeleton, `go.mod`, Makefile, `.gitignore`.
- Domain types + ports compiling (`internal/domain`, `internal/port`).
- `docker-compose` up: all chain nodes + Kafka.

### ✓ Day 2 — Walking skeleton (EVM end-to-end)
- `evm` adapter: derive account (secp256k1 BIP-44), balance, EIP-1559 build/sign/broadcast, get tx.
- `KeyStore` (Argon2id + AES-256-GCM) + in-memory `Signer`; in-memory `WalletRepository`.
- `WalletService` + `TransactionService` create/balance/transfer implemented.
- `chi` HTTP transport. Config loader (env → RPC URL/chainId).
- `turba/DESIGN.md`: traffic simulator design.
- **Demo verified: create wallet → fund from Anvil → transfer 0.1 ETH → confirmed on-chain.**

### ✓ Day 3 — Second surface + second family
- CLI transport over the same services (proves NFR: one core, two surfaces).
- `bitcoin` adapter: P2PKH derivation, scantxoutset balance, largest-first UTXO coin
  selection with coinbase maturity filter, P2PKH scriptSig, sendrawtransaction.
- **Demo verified: derive BTC address → mine 101 regtest blocks → transfer 1 BTC → confirmed.**

### ✓ Day 4 — Solana
- `solana` adapter: SLIP-0010 ed25519 derivation (`m/44'/501'/0'/0'`), balance, EIP-style fee estimate (5000 lamports), build/sign/broadcast, get tx.
- `DeriveKeyEd25519` added to `hdwallet`; keystore signer dispatches to ed25519 for Solana.
- **Demo verified: airdrop 2 SOL → balance → transfer 0.1 SOL → confirmed on-chain (test-validator).**

### Day 5 — TRON + Base
- `tron` adapter (TVM, base58 addresses, energy/bandwidth).
- Wire **Base as a second EVM config** (alt chainId/endpoint) — minimal new code.

### Day 6 — Monitoring
- `ChainWatcher` per chain (polling), `MonitorService`.
- Kafka `TxEventPublisher`; events flowing to a topic.

### Day 7 — Hardening & polish
- Fee estimate endpoints (naive per chain).
- Tests for domain + services (adapters mocked).
- README usage examples, end-to-end smoke across all five chains on `local`.

> Sequencing note: TRON's local node is the heaviest — if it stalls, fall back
> to its testnet (Shasta/Nile) for that chain without blocking the rest.

## V2 — beyond the week

- **Cross-chain interoperability / bridging** between supported chains.
- **Custom token support** (ERC-20 / SPL / TRC-20) — extends the existing
  `Asset` + `ChainAdapter` seams.
- **Blockchain traffic & cost monitoring** and **improved fee estimation** —
  historical fee-market sampling behind the existing `ChainWatcher` /
  `FeeEstimate` seams (see [ADR-0008](./adr/0008-fee-and-monitoring-seams.md)).
- Testnet/mainnet promotion via config (no code changes).
