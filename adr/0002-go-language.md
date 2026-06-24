# ADR-0002: Go as the implementation language

- **Status:** Accepted
- **Date:** 2026-06-24

## Context

Candidates were Go, Rust, and Java. Selection criteria: blockchain library
support across all five chains, fit for a one-week MVP exposing both an API and
a CLI plus Kafka, and job-market signal for a portfolio piece.

Library landscape:
- **Bitcoin:** Go `btcd`/btcsuite (mature); Rust `rust-bitcoin`/BDK (best); Java
  `bitcoinj` (dated).
- **Ethereum/Base:** Go `go-ethereum` (canonical); Rust `alloy`; Java `web3j`.
- **Solana:** Go `solana-go` (workable); Rust (official, best); Java (weak).
- **TRON:** Go `gotron-sdk`; Rust (thin); Java (java-tron is the reference node).

## Decision

Use **Go**. It has canonical-quality BTC/EVM libraries, workable Solana/TRON
support, a trivial single-binary CLI+API story, first-class concurrency for the
watcher/Kafka work, and broad employability across backend and crypto-infra.

## Consequences

- **+** Best effort-to-result ratio for five adapters in a week.
- **+** Strong concurrency model suits transaction monitoring.
- **−** Solana/TRON libraries are less mature than Rust's; expect rough edges.
- **−** Forgoes Rust's stronger crypto-native hiring signal — acceptable given
  the breadth and timeline goals. Revisit if target clients are crypto-native.
