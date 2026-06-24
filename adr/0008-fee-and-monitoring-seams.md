# ADR-0008: Fee-estimation and monitoring seams designed in V1 for V2

- **Status:** Accepted
- **Date:** 2026-06-24

## Context

V2 will add **blockchain traffic & cost monitoring** and **better transaction
price estimation**. If fee estimation and monitoring are bolted on later, they
risk forcing a redesign. They should be seams from the start.

## Decision

- Include `EstimateFee` in the `ChainAdapter` port from V1, returning
  `domain.FeeEstimate` with a coarse `FeeSpeed` hint and a `Params` map for
  chain-specific detail (gas price+limit, sat/vB, compute units, energy). V1
  implementations may be naive (e.g. node-suggested values).
- Keep transaction monitoring behind the `ChainWatcher` port so V2 can swap in
  implementations that also sample fee markets and emit traffic/cost telemetry.
- Treat RPC access as instrumentable so per-call metrics can be added without
  touching callers (NFR6).

## Consequences

- **+** V2 improved estimation = better `ChainWatcher`/`FeeEstimate`
  implementations behind unchanged signatures.
- **+** Historical fee/traffic data can be collected by a future adapter and
  persisted via a new repository without disturbing the core.
- **−** Carrying `Params` as `map[string]string` is loosely typed; chain
  adapters own the contract for their keys. Revisit if it becomes error-prone.
