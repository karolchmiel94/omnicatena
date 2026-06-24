package domain

import "math/big"

type Asset struct {
	Symbol   string
	Decimals uint8
	Native   bool
	// ContractAddress is the seam for V2 token support (ERC-20/SPL/TRC-20); empty for native assets.
	ContractAddress string
}

type Amount struct {
	Asset Asset
	// Base is the amount in the asset's smallest unit (satoshi, wei, lamport, sun),
	// as big.Int to avoid floating-point drift.
	Base *big.Int
}
