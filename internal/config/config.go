package config

import "os"

type EVM struct {
	RPCURL  string
	ChainID int64
}

type Bitcoin struct {
	Host string // host:port
	User string
	Pass string
}

type Config struct {
	Ethereum EVM
	Base     EVM
	Bitcoin  Bitcoin
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
		Bitcoin: Bitcoin{
			Host: getenv("BITCOIN_RPC_HOST", "localhost:18443"),
			User: getenv("BITCOIN_RPC_USER", "omni"),
			Pass: getenv("BITCOIN_RPC_PASS", "omni"),
		},
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
