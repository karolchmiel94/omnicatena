package unit_test

import (
	"testing"
	"time"

	"github.com/karolchmiel94/omnicatena/internal/domain"
)

func TestWallet_Account_Found(t *testing.T) {
	w := domain.Wallet{
		ID:        "w1",
		Label:     "test",
		CreatedAt: time.Now(),
		Accounts: []domain.Account{
			{Chain: domain.ChainEthereum, Address: "0xabc", Path: "m/44'/60'/0'/0/0"},
			{Chain: domain.ChainBitcoin, Address: "1abc", Path: "m/44'/0'/0'/0/0"},
		},
	}

	acc, ok := w.Account(domain.ChainEthereum)
	if !ok {
		t.Fatal("Account(ChainEthereum): expected found=true")
	}
	if acc.Address != "0xabc" {
		t.Errorf("Address: got %q, want %q", acc.Address, "0xabc")
	}
	if acc.Chain != domain.ChainEthereum {
		t.Errorf("Chain: got %s, want %s", acc.Chain, domain.ChainEthereum)
	}
}

func TestWallet_Account_NotFound(t *testing.T) {
	w := domain.Wallet{
		Accounts: []domain.Account{
			{Chain: domain.ChainEthereum},
		},
	}
	_, ok := w.Account(domain.ChainSolana)
	if ok {
		t.Error("Account(ChainSolana): expected found=false for missing chain")
	}
}

func TestWallet_Account_EmptyWallet(t *testing.T) {
	w := domain.Wallet{}
	_, ok := w.Account(domain.ChainEthereum)
	if ok {
		t.Error("Account on empty wallet: expected found=false")
	}
}

func TestWallet_Account_ReturnsFirstMatch(t *testing.T) {
	// If somehow two accounts share the same chain, we get the first.
	w := domain.Wallet{
		Accounts: []domain.Account{
			{Chain: domain.ChainEthereum, Address: "0xfirst"},
			{Chain: domain.ChainEthereum, Address: "0xsecond"},
		},
	}
	acc, ok := w.Account(domain.ChainEthereum)
	if !ok {
		t.Fatal("expected found=true")
	}
	if acc.Address != "0xfirst" {
		t.Errorf("expected first match, got %s", acc.Address)
	}
}
