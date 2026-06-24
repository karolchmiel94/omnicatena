// Package evm implements the chain ports for EVM chains. One instance serves
// Ethereum and another serves Base, differing only by config (chainId, endpoint,
// gas), not separate code (ADR-0007). Built on go-ethereum.
package evm
