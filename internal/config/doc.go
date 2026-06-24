// Package config loads runtime config and maps each domain.Network onto a
// concrete RPC endpoint and chain parameters. It is the single place that knows
// about endpoints, keeping "network is configuration, not code" true (ADR-0003).
package config
