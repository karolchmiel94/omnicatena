package unit_test

import (
	"strings"
	"testing"

	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/evm"
	"github.com/karolchmiel94/omnicatena/internal/domain"
	bip39 "github.com/tyler-smith/go-bip39"
)

func newBaseAdapter(t *testing.T) *evm.Adapter {
	t.Helper()
	a, err := evm.New(evm.Config{
		RPCURL:  "http://localhost:8546",
		ChainID: 8453,
		Chain:   domain.ChainBase,
	})
	if err != nil {
		t.Fatalf("evm.New (base): %v", err)
	}
	return a
}

func TestBaseAdapter_Chain(t *testing.T) {
	a := newBaseAdapter(t)
	if a.Chain() != domain.ChainBase {
		t.Errorf("Chain() = %s, want %s", a.Chain(), domain.ChainBase)
	}
}

func TestBaseAdapter_DeriveAccount_Deterministic(t *testing.T) {
	a := newBaseAdapter(t)
	seed := bip39.NewSeed("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "")
	path := domain.DerivationPath("m/44'/60'/0'/0/0")

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
	if acc1.Chain != domain.ChainBase {
		t.Errorf("chain: got %s, want %s", acc1.Chain, domain.ChainBase)
	}
	if acc1.Path != path {
		t.Errorf("path not preserved: got %s, want %s", acc1.Path, path)
	}
}

func TestBaseAdapter_DeriveAccount_AddressFormat(t *testing.T) {
	a := newBaseAdapter(t)
	seed := bip39.NewSeed("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "")
	acc, err := a.DeriveAccount(seed, "m/44'/60'/0'/0/0")
	if err != nil {
		t.Fatalf("DeriveAccount: %v", err)
	}
	addr := string(acc.Address)
	if !strings.HasPrefix(addr, "0x") {
		t.Errorf("Base address should start with 0x, got %s", addr)
	}
	if len(addr) != 42 {
		t.Errorf("Base address should be 42 chars, got %d (%s)", len(addr), addr)
	}
}

func TestBaseAdapter_DeriveAccount_SameCoinTypeAsEthereum(t *testing.T) {
	// Base uses coin type 60, same as Ethereum (ADR-0007): same seed+path → same
	// 20-byte address on both chains. Only Account.Chain differs.
	seed := bip39.NewSeed("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "")
	path := domain.DerivationPath("m/44'/60'/0'/0/0")

	ethAdapter, err := evm.New(evm.Config{RPCURL: "http://localhost:8545", ChainID: 31337, Chain: domain.ChainEthereum})
	if err != nil {
		t.Fatalf("eth adapter: %v", err)
	}
	baseAdapter, err := evm.New(evm.Config{RPCURL: "http://localhost:8546", ChainID: 8453, Chain: domain.ChainBase})
	if err != nil {
		t.Fatalf("base adapter: %v", err)
	}

	ethAcc, err := ethAdapter.DeriveAccount(seed, path)
	if err != nil {
		t.Fatalf("eth DeriveAccount: %v", err)
	}
	baseAcc, err := baseAdapter.DeriveAccount(seed, path)
	if err != nil {
		t.Fatalf("base DeriveAccount: %v", err)
	}

	if ethAcc.Address != baseAcc.Address {
		t.Errorf("expected same address for coin type 60: eth=%s base=%s", ethAcc.Address, baseAcc.Address)
	}
	if ethAcc.Chain != domain.ChainEthereum {
		t.Errorf("eth chain: got %s", ethAcc.Chain)
	}
	if baseAcc.Chain != domain.ChainBase {
		t.Errorf("base chain: got %s", baseAcc.Chain)
	}
}
