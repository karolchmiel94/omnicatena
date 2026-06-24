# ADR-0001: Hexagonal (ports & adapters) architecture

- **Status:** Accepted
- **Date:** 2026-06-24

## Context

The system must present five heterogeneous blockchains (Bitcoin, Ethereum, Base,
Solana, TRON) behind common `Wallet` and `Transaction` abstractions, and must
keep datastore, message broker, RPC clients, and key storage swappable. There
are five real adapter implementations on day one, so the variation point is
concrete — not speculative abstraction.

## Decision

Adopt ports & adapters. A pure `domain` and an `app` (use-case) layer depend
only on interfaces (`port`). All technology-specific code lives in `adapter`
(driven) and `transport` (driving) packages. The dependency rule points inward.

## Consequences

- **+** Adding a chain = one new adapter + registry entry; core untouched (NFR1).
- **+** Domain/use cases are unit-testable with mocked ports (NFR5).
- **+** Kafka, datastore, key storage, RPC are all replaceable/stubbable (NFR2).
- **−** More indirection and boilerplate than a flat design — accepted because
  the variation is real and this is partly a demonstration of the pattern.
- Abstraction is applied **only** at named variation points; everything else
  stays simple.
