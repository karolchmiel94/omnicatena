# ADR-0006: Unsigned-transaction envelope

- **Status:** Accepted
- **Date:** 2026-06-24

## Context

We want a uniform `Transaction` abstraction, but the unsigned transaction is
irreconcilably different per chain: UTXO inputs/outputs + coin selection
(Bitcoin), nonce + gas (EVM), recent-blockhash + fee payer (Solana), TVM +
energy/bandwidth (TRON). Forcing a single universal transaction struct would
either leak chain details into the core or lose fidelity.

## Decision

Make the **lifecycle** uniform in the `ChainAdapter` port (`EstimateFee →
BuildTransfer → Sign → Broadcast → GetTransaction`), and carry the unsigned
transaction as an **envelope**: `domain.UnsignedTx` wraps a chain-specific,
opaque `Payload []byte`. The application layer never inspects `Payload`; only
the adapter that built it can sign it.

## Consequences

- **+** Generic use cases over heterogeneous transactions; no chain leakage.
- **+** Each adapter is free to use whatever internal representation its chain
  and library require.
- **−** `Payload` is opaque to the core, so cross-chain logic that needs to
  understand transaction internals (e.g. V2 bridging) must do so inside
  adapters or a dedicated interop layer, not the generic core.
