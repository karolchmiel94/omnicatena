package config

import (
	"os"
	"strings"
)

type EVM struct {
	RPCURL  string
	ChainID int64
}

type Bitcoin struct {
	Host string // host:port
	User string
	Pass string
}

type Solana struct {
	RPCURL string
}

type Tron struct {
	RPCURL string
}

// Kafka publishing is optional (ADR-0005) — disabled by default so running the
// API doesn't require a broker. Enable with KAFKA_ENABLED=true.
type Kafka struct {
	Enabled bool
	Brokers []string
	Topic   string
}

type Database struct {
	URL string
}

type Config struct {
	Ethereum EVM
	Base     EVM
	Bitcoin  Bitcoin
	Solana   Solana
	Tron     Tron
	Kafka    Kafka
	Database Database
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
		Solana: Solana{
			RPCURL: getenv("SOLANA_RPC_URL", "http://localhost:8899"),
		},
		Tron: Tron{
			RPCURL: getenv("TRON_RPC_URL", "http://localhost:9090"),
		},
		Kafka: Kafka{
			Enabled: getenv("KAFKA_ENABLED", "false") == "true",
			Brokers: strings.Split(getenv("KAFKA_BROKERS", "localhost:9092"), ","),
			Topic:   getenv("KAFKA_TOPIC", "omnicatena.tx.events"),
		},
		Database: Database{
			URL: getenv("DATABASE_URL", "postgres://omni:omni@localhost:5433/omnicatena?sslmode=disable"),
		},
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
