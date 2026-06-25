package config

import "os"

type EVM struct {
	RPCURL  string
	ChainID int64
}

type Config struct {
	Ethereum EVM
	Base     EVM
}

func Load() Config {
	return Config{
		Ethereum: EVM{
			RPCURL:  getenv("ETHEREUM_RPC_URL", "http://localhost:8545"),
			ChainID: 31337,
		},
		Base: EVM{
			RPCURL:  getenv("BASE_RPC_URL", "http://localhost:8546"),
			ChainID: 8453,
		},
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
