# Requirements

## Context

- **Type:** greenfield B2C prototype / portfolio project; single developer.
- **Deployment:** local machine only. No cloud, no multi-tenant concerns.
- **Failure tolerance:** prototype — breakage is acceptable, no SLA.
- **Goal of V1:** a working, well-architected multi-chain wallet usable via API
  and CLI, deliverable in ~1 week.

## Functional requirements (V1)

Prioritized with MoSCoW.

### Must
- **F1 — Create wallet.** Generate a new wallet with a single seed and derive
  one account per supported chain (BTC, ETH, Base, SOL, TRON).
- **F2 — List / get wallet.** Enumerate wallets and view a wallet's accounts
  (addresses per chain).
- **F3 — Get balance.** Query the native-asset balance of a wallet's account on
  a given chain.
- **F4 — Transfer native value.** Build → sign → broadcast a native transfer on
  any supported chain, returning a transaction hash.
- **F5 — Transaction status.** Look up the on-chain status of a transaction by
  hash (pending / confirmed / failed, confirmations).
- **F6 — Monitor transactions.** Watch wallet addresses and publish on-chain
  events (inbound/outbound/confirmed) to a Kafka topic.
- **F7 — Two delivery surfaces.** Every capability above is reachable from both
  the HTTP API and the CLI, sharing one application core.

### Should
- **F8 — Fee estimate.** Return an estimated fee for a pending transfer
  (naive per-chain implementation in V1; see [ADR-0008](./adr/0008-fee-and-monitoring-seams.md)).
- **F9 — Network selection.** Target `local`, `testnet`, or `mainnet` per chain
  via configuration without code changes ([ADR-0003](./adr/0003-network-as-configuration.md)).

### Could
- Multiple accounts/addresses per chain within one wallet (HD account index).
- Address book / labels.

### Won't (V1 non-goals)
- **Token transfers** (ERC-20 / SPL / TRC-20) — *port is designed for it; impl is V2.*
- **Smart-contract / token deployment** — listed in early discovery notes;
  deferred to V2 to protect the one-week target ([ADR-0009](./adr/0009-v1-scope-native-transfers-only.md)).
- **Cross-chain interoperability / bridging** — V2.
- **Rich traffic & cost analytics** — V2.
- Multi-tenant, auth/authz, user accounts.
- Cloud deployment, HA, horizontal scaling.
- Production key custody (HSM, MPC, hardware wallets).
- A web or mobile UI.

## Non-functional requirements

- **NFR1 — Extensibility.** Adding a new chain = adding one adapter implementing
  the `ChainAdapter` port, with **zero changes to the domain or use cases**.
- **NFR2 — Swappability.** Datastore, message broker, RPC clients, and key
  storage are all adapters behind ports and can be replaced or stubbed.
- **NFR3 — Security of key material.** Seeds are encrypted at rest (Argon2 +
  AES-GCM). Plaintext seeds/keys never cross the application boundary and are
  never logged ([ADR-0004](./adr/0004-hd-wallet-key-management.md)).
- **NFR4 — Determinism for local dev.** The whole stack (nodes + Kafka) comes up
  via `docker compose`, self-funded, requiring no third-party API keys.
- **NFR5 — Testability.** Domain and application layers are testable without a
  live chain (adapters mocked at the port).
- **NFR6 — Observability seam.** RPC calls and watcher activity are designed to
  emit metrics, enabling V2 traffic/cost monitoring.
- **NFR7 — Read throughput.** The API layer targets ~2000 req/s for *local/read*
  operations (balances, wallet lookups, cached state). On-chain writes are bounded
  by the chains themselves and are explicitly out of this target.

## V2 (recorded now to inform design, not built in V1)

- **Cross-chain interoperability** between the supported chains.
- **Custom token support** across chains (transfers).
- **Smart-contract / token deployment** — modeled as an optional
  `ContractDeployer` capability interface that only chains supporting it
  implement (EVM/Solana/TRON; Bitcoin opts out), keeping `ChainAdapter` minimal.
- **Blockchain traffic & cost monitoring** and **improved fee estimation** —
  historical sampling of gas/fee markets to price transactions better.
