package domain

type ChainID string

const (
	ChainBitcoin  ChainID = "bitcoin"
	ChainEthereum ChainID = "ethereum"
	ChainBase     ChainID = "base"
	ChainSolana   ChainID = "solana"
	ChainTron     ChainID = "tron"
)

type NetworkEnv string

const (
	EnvLocal   NetworkEnv = "local"
	EnvTestnet NetworkEnv = "testnet"
	EnvMainnet NetworkEnv = "mainnet"
)

// The Network→endpoint/params mapping lives in configuration, not adapter code (ADR-0003).
type Network struct {
	Chain ChainID
	Env   NetworkEnv
}
