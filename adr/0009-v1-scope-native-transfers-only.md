# ADR-0009: V1 scope is native transfers only

- **Status:** Accepted
- **Date:** 2026-06-24

## Context

Early discovery notes listed token transfers and **smart-contract / token
deployment** as V1 scope. Subsequent live scoping chose native transfers only,
with tokens in V2. The two needed reconciling before building.

## Decision

V1 ships **native value transfers** (create wallet, balance, send/receive,
status, monitoring) across all five chains, plus a naive fee estimate. **Token
transfers and contract/token deployment are V2.** Rationale: a solo, one-week,
five-chain target is already ambitious; adding deployment (another vertical per
capable chain) and token transfers would jeopardize it.

When built in V2, deployment will be an **optional `ContractDeployer` capability
interface** that only chains supporting it implement (EVM/Solana/TRON); Bitcoin
opts out. This keeps the core `ChainAdapter` minimal and avoids forcing a
meaningless method onto UTXO chains.

The "2000 req/s" figure from the notes is scoped to the **API read path**, not
on-chain throughput (NFR7).

## Consequences

- **+** Protects the one-week target; clean walking skeleton focused on transfers.
- **+** Deployment/token seams are designed (capability interface + `Asset`
  contract address) so V2 is additive, not a redesign.
- **−** The shipped V1 is narrower than the original written scope; this ADR is
  the record of that deliberate trade-off.
