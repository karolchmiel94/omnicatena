# ADR-0005: Kafka behind a `TxEventPublisher` port

- **Status:** Accepted
- **Date:** 2026-06-24

## Context

Transaction monitoring should emit events to Kafka. For a single-dev local
prototype Kafka is arguably over-engineering on merits, but demonstrating an
event-driven monitoring pipeline is an explicit learning/portfolio goal.

## Decision

Keep Kafka, but place it behind a `TxEventPublisher` port. A `ChainWatcher`
detects on-chain activity and the `MonitorService` publishes `domain.TxEvent`s
through the port. The Kafka adapter is one implementation; in-memory and stdout
publishers serve tests and lightweight local runs.

## Consequences

- **+** Satisfies the event-driven goal without coupling the core to Kafka.
- **+** Tests and demos can run without a broker.
- **−** Operational weight of running Kafka locally (mitigated: single-node
  KRaft via Docker, no ZooKeeper).
