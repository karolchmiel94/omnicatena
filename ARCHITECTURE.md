# Architecture

Omnicatena uses **ports & adapters (hexagonal architecture)**. The justification
is concrete, not aspirational: five blockchains must sit behind common `Wallet`
and `Transaction` abstractions, and the datastore / broker / RPC / key storage
must all be swappable ([ADR-0001](./adr/0001-hexagonal-architecture.md)).

## The hexagon

```
                 Driving (inbound) adapters
            ┌───────────────┐   ┌───────────────┐
            │   HTTP API     │   │      CLI      │
            └───────┬───────┘   └───────┬───────┘
                    │   (same use cases)│
              ┌─────▼───────────────────▼─────┐
              │      Application services      │   internal/app
              │  WalletService, TxService,     │
              │  MonitorService                │
              ├────────────────────────────────┤
              │            Domain              │   internal/domain
              │  Wallet, Account, Transaction, │   (pure, no I/O)
              │  Amount, Asset, Network, Fee   │
              └─────┬───────────┬───────┬──────┘
                    │ ports     │       │
        ┌───────────▼──┐ ┌──────▼────┐ ┌▼────────────────┐
        │ ChainAdapter │ │ KeyStore  │ │ TxEventPublisher│   internal/port
        │ ChainWatcher │ │ WalletRepo│ │ Signer          │
        └──────┬───────┘ └─────┬─────┘ └────────┬────────┘
               │               │                │
       Driven (outbound) adapters  →  internal/adapter
   ┌────┬──────┬───────┬──────┐ ┌──────────┐ ┌────────────┐
   │EVM │Bitcoin│Solana│ TRON │ │ keystore │ │ Kafka      │
   │(ETH+Base) │      │      │ │ + repo   │ │ publisher  │
   └────┴──────┴───────┴──────┘ └──────────┘ └────────────┘
```

**Dependency rule:** arrows point inward. `domain` imports nothing from the
project. `app` depends on `domain` and `port` (interfaces only). `adapter` and
the transports depend on `app`/`port`/`domain`, never the reverse.

## Why these seams

- **`ChainAdapter`** — one per chain *family*. The EVM adapter serves **both
  Ethereum and Base** via config (different chainId + endpoint), not duplicated
  code ([ADR-0007](./adr/0007-evm-serves-ethereum-and-base.md)). Adding a chain
  is a new adapter + a registry entry; the core is untouched (NFR1).
- **`ChainWatcher`** — monitoring seam. V1 polls; V2 swaps in richer
  traffic/cost telemetry without touching callers ([ADR-0008](./adr/0008-fee-and-monitoring-seams.md)).
- **`TxEventPublisher`** — Kafka is an adapter, so monitoring can run with Kafka
  in Docker, or with an in-memory/stdout publisher in tests
  ([ADR-0005](./adr/0005-kafka-behind-port.md)).
- **`KeyStore` / `Signer`** — key material is encrypted at rest and brokered
  through `Signer`; plaintext keys never reach the application layer
  ([ADR-0004](./adr/0004-hd-wallet-key-management.md)).
- **`WalletRepository`** — wallet *metadata* persistence (never secrets).

## The hard part: the unsigned-transaction envelope

The transaction lifecycle is uniform (`estimateFee → buildTransfer → sign →
broadcast → getTransaction`) and lives in the port. But the *unsigned
transaction itself* is irreconcilably different across chains — UTXO coin
selection vs. EVM nonce+gas vs. Solana recent-blockhash+fee-payer.

We do **not** force a universal transaction struct. Instead, `domain.UnsignedTx`
is an **envelope**: a uniform wrapper carrying a chain-specific, opaque
`Payload []byte`. The application layer never inspects `Payload`; only the
adapter that produced it can sign it. This keeps the lifecycle generic while
letting each chain do whatever it must internally
([ADR-0006](./adr/0006-unsigned-tx-envelope.md)).

## Directory layout

```
omnicatena/
├── cmd/
│   ├── api/            # composition root for the HTTP API
│   └── cli/            # composition root for the CLI
├── internal/
│   ├── domain/         # entities & value objects — pure, no I/O
│   ├── port/           # all interface definitions (the hexagon's edges)
│   ├── app/            # application services (use cases)
│   ├── adapter/        # driven adapters
│   │   ├── chain/
│   │   │   ├── evm/        # Ethereum + Base
│   │   │   ├── bitcoin/
│   │   │   ├── solana/
│   │   │   └── tron/
│   │   ├── keystore/      # encrypted seed storage + Signer
│   │   ├── repository/    # wallet metadata persistence
│   │   └── events/
│   │       └── kafka/     # TxEventPublisher over Kafka
│   ├── transport/      # driving adapters
│   │   ├── http/
│   │   └── cli/
│   └── config/         # config loading; maps networks → endpoints
├── deployments/docker/ # node + Kafka compose assets
├── adr/                # Architecture Decision Records
└── docker-compose.yml
```

## Key flows

**Create wallet (F1):** `WalletService.Create` → `KeyStore.Create` (generate +
encrypt seed) → for each supported chain, `ChainAdapter.DeriveAccount(seed,
path)` → persist `Wallet` metadata via `WalletRepository`.

**Transfer (F4):** `TxService.Transfer` → `ChainAdapter.EstimateFee` →
`BuildTransfer` (opaque `UnsignedTx`) → `Sign` (via `Signer` over an unlocked
seed) → `Broadcast` → return hash.

**Monitor (F6):** `MonitorService` starts a `ChainWatcher` per chain → events
fan into `TxEventPublisher` (Kafka).

## Optional capabilities (V2-ready)

Not every chain supports every capability — e.g. smart-contract/token
deployment exists on EVM/Solana/TRON but not meaningfully on Bitcoin. Rather
than bloat `ChainAdapter` with methods some chains can't honor, V2 capabilities
are **optional interfaces** an adapter may additionally implement, e.g.:

```go
type ContractDeployer interface {
    DeployContract(ctx context.Context, req DeployRequest) (Deployment, error)
}
```

Callers type-assert (`if d, ok := adapter.(ContractDeployer); ok { ... }`) and
degrade gracefully when a chain doesn't support it. This keeps the core port
minimal while letting capable chains expose more — the same pattern will host
token transfers and richer monitoring (see ADR-0009, ADR-0008).
