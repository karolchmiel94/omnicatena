# ADR-0007: One EVM adapter serves Ethereum and Base

- **Status:** Accepted
- **Date:** 2026-06-24

## Context

Base is an Ethereum L2: same address format, same secp256k1 signing, same
transaction structure as Ethereum. The five chains are really **four adapter
families** (UTXO, EVM, Solana, TRON).

## Decision

Implement a single `evm` adapter and instantiate it twice — once for Ethereum,
once for Base — differing only by configuration (chainId, RPC endpoint, gas
settings). Do **not** write a separate Base adapter; that would be the first
unearned abstraction.

## Consequences

- **+** Base support is essentially free once EVM works.
- **+** Future EVM chains (Arbitrum, Optimism, Polygon) are config entries.
- **Local caveat:** running a real local Base (OP-stack) node is too heavy, so
  locally Base is a second Anvil instance with a distinct chainId; on testnet,
  `base` targets real Base Sepolia (see [ADR-0003](./0003-network-as-configuration.md)).
- **−** EVM-chain-specific quirks (e.g. L2 fee components) must be handled by
  config/branching inside the one adapter rather than separate types.
