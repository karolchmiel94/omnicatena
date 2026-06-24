# ADR-0003: Network is configuration, not code

- **Status:** Accepted
- **Date:** 2026-06-24

## Context

V1 targets **local nodes** first, with testnets later and mainnet a possible
future. The same adapter logic must talk to any of these without code changes.

## Decision

Model the target as `domain.Network = {Chain, Env}` where `Env ∈ {local,
testnet, mainnet}`. Config maps each `(chain, env)` to an RPC endpoint and chain
parameters (chainId, address prefixes, etc.). Adapters receive their endpoint and
params via config; they contain no hardcoded network assumptions.

## Consequences

- **+** Promote from local → testnet → mainnet by editing config only (F9).
- **+** Local default needs no third-party API keys (NFR4).
- **−** Slightly more config surface and validation up front.
- **Local Base note:** a real local Base node (OP-stack) is too heavy for this
  project; locally Base is simulated as a second EVM/Anvil instance with a
  distinct chainId. On testnet, `base` points at real Base Sepolia. See
  [ADR-0007](./0007-evm-serves-ethereum-and-base.md).
