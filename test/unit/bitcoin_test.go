package unit_test

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/bitcoin"
	"github.com/karolchmiel94/omnicatena/internal/domain"
	bip39 "github.com/tyler-smith/go-bip39"
)

// newRegtestAdapter creates a Bitcoin adapter configured for regtest.
// rpcclient.New in HTTP mode does not dial at construction time, so no
// bitcoind node needs to be running.
func newRegtestAdapter(t *testing.T) *bitcoin.Adapter {
	t.Helper()
	a, err := bitcoin.New(bitcoin.Config{
		Host:        "localhost:18443",
		User:        "omni",
		Pass:        "omni",
		ChainParams: &chaincfg.RegressionNetParams,
	})
	if err != nil {
		t.Fatalf("bitcoin.New: %v", err)
	}
	return a
}

func abandonSeed() []byte {
	const mnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	return bip39.NewSeed(mnemonic, "")
}

func TestBitcoinAdapter_DeriveAccount_Deterministic(t *testing.T) {
	a := newRegtestAdapter(t)
	seed := abandonSeed()
	path := domain.DerivationPath("m/44'/0'/0'/0/0")

	acc1, err := a.DeriveAccount(seed, path)
	if err != nil {
		t.Fatalf("DeriveAccount: %v", err)
	}
	acc2, err := a.DeriveAccount(seed, path)
	if err != nil {
		t.Fatal(err)
	}
	if acc1.Address != acc2.Address {
		t.Errorf("non-deterministic: %s != %s", acc1.Address, acc2.Address)
	}
	if acc1.Chain != domain.ChainBitcoin {
		t.Errorf("chain: got %s, want %s", acc1.Chain, domain.ChainBitcoin)
	}
	if acc1.Path != path {
		t.Errorf("path not preserved: got %s, want %s", acc1.Path, path)
	}
	if len(acc1.Address) == 0 {
		t.Error("address is empty")
	}
}

func TestBitcoinAdapter_DeriveAccount_DifferentIndices(t *testing.T) {
	a := newRegtestAdapter(t)
	seed := abandonSeed()

	acc0, err := a.DeriveAccount(seed, "m/44'/0'/0'/0/0")
	if err != nil {
		t.Fatal(err)
	}
	acc1, err := a.DeriveAccount(seed, "m/44'/0'/0'/0/1")
	if err != nil {
		t.Fatal(err)
	}
	if acc0.Address == acc1.Address {
		t.Error("different path indices should produce different addresses")
	}
}

func TestBitcoinAdapter_DeriveAccount_RegtestAddressFormat(t *testing.T) {
	// Regtest P2PKH addresses use the same version byte as testnet (0x6f),
	// which encodes to base58check addresses starting with 'm' or 'n'.
	a := newRegtestAdapter(t)
	acc, err := a.DeriveAccount(abandonSeed(), "m/44'/0'/0'/0/0")
	if err != nil {
		t.Fatalf("DeriveAccount: %v", err)
	}
	first := string(acc.Address)[0]
	if first != 'm' && first != 'n' {
		t.Errorf("regtest P2PKH address should start with m or n, got %c (addr: %s)", first, acc.Address)
	}
}
