package unit_test

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/tron"
	"github.com/karolchmiel94/omnicatena/internal/domain"
	bip39 "github.com/tyler-smith/go-bip39"
)

func newTronAdapter() *tron.Adapter {
	return tron.New(tron.Config{RPCURL: "http://localhost:9090"})
}

func abandonSeedTron() []byte {
	return bip39.NewSeed("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "")
}

func TestTronAdapter_Chain(t *testing.T) {
	if newTronAdapter().Chain() != domain.ChainTron {
		t.Errorf("Chain() = %s, want %s", newTronAdapter().Chain(), domain.ChainTron)
	}
}

func TestTronAdapter_DeriveAccount_Deterministic(t *testing.T) {
	a := newTronAdapter()
	seed := abandonSeedTron()
	path := domain.DerivationPath("m/44'/195'/0'/0/0")

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
	if acc1.Chain != domain.ChainTron {
		t.Errorf("chain: got %s, want %s", acc1.Chain, domain.ChainTron)
	}
	if acc1.Path != path {
		t.Errorf("path not preserved: got %s, want %s", acc1.Path, path)
	}
}

func TestTronAdapter_DeriveAccount_DifferentIndices(t *testing.T) {
	a := newTronAdapter()
	seed := abandonSeedTron()

	acc0, _ := a.DeriveAccount(seed, "m/44'/195'/0'/0/0")
	acc1, _ := a.DeriveAccount(seed, "m/44'/195'/0'/0/1")
	if acc0.Address == acc1.Address {
		t.Error("different path indices should produce different addresses")
	}
}

func TestTronAdapter_DeriveAccount_AddressFormat(t *testing.T) {
	acc, err := newTronAdapter().DeriveAccount(abandonSeedTron(), "m/44'/195'/0'/0/0")
	if err != nil {
		t.Fatalf("DeriveAccount: %v", err)
	}
	addr := string(acc.Address)
	if !strings.HasPrefix(addr, "T") {
		t.Errorf("TRON address should start with T, got %s", addr)
	}
	// base58check(0x41 + 20 bytes) is always 34 chars
	if len(addr) != 34 {
		t.Errorf("TRON address should be 34 chars, got %d (%s)", len(addr), addr)
	}
}

func TestTronAdapter_EstimateFee_BandwidthCeiling(t *testing.T) {
	fee, err := newTronAdapter().EstimateFee(context.Background(), domain.TransferRequest{Speed: domain.SpeedStandard})
	if err != nil {
		t.Fatalf("EstimateFee: %v", err)
	}
	want := big.NewInt(268_000)
	if fee.Total.Base.Cmp(want) != 0 {
		t.Errorf("total: got %s sun, want %s", fee.Total.Base, want)
	}
	if fee.Params["bandwidth_bytes"] != "268" {
		t.Errorf("bandwidth_bytes: got %q, want 268", fee.Params["bandwidth_bytes"])
	}
	if fee.Params["bandwidth_price"] != "1000" {
		t.Errorf("bandwidth_price: got %q, want 1000", fee.Params["bandwidth_price"])
	}
}

func TestTronAdapter_EstimateFee_SpeedPreserved(t *testing.T) {
	a := newTronAdapter()
	for _, speed := range []domain.FeeSpeed{domain.SpeedEconomy, domain.SpeedStandard, domain.SpeedFast} {
		fee, err := a.EstimateFee(context.Background(), domain.TransferRequest{Speed: speed})
		if err != nil {
			t.Fatalf("EstimateFee(%s): %v", speed, err)
		}
		if fee.Speed != speed {
			t.Errorf("Speed: got %s, want %s", fee.Speed, speed)
		}
	}
}

func TestTronAdapter_EstimateFee_FlatRegardlessOfSpeed(t *testing.T) {
	// Bandwidth cost doesn't scale with speed — all speeds return the same ceiling.
	a := newTronAdapter()
	eco, _ := a.EstimateFee(context.Background(), domain.TransferRequest{Speed: domain.SpeedEconomy})
	fast, _ := a.EstimateFee(context.Background(), domain.TransferRequest{Speed: domain.SpeedFast})
	if eco.Total.Base.Cmp(fast.Total.Base) != 0 {
		t.Errorf("fee should be flat across speeds: economy=%s fast=%s", eco.Total.Base, fast.Total.Base)
	}
}
