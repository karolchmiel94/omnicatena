package unit_test

import (
	"context"
	"testing"

	chainreg "github.com/karolchmiel94/omnicatena/internal/adapter/chain"
	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/port"
)

type stubAdapter struct{ chain domain.ChainID }

func (s *stubAdapter) Chain() domain.ChainID { return s.chain }
func (s *stubAdapter) DeriveAccount(_ []byte, _ domain.DerivationPath) (domain.Account, error) {
	return domain.Account{}, nil
}
func (s *stubAdapter) Balance(_ context.Context, _ domain.Address, _ domain.Asset) (domain.Amount, error) {
	return domain.Amount{}, nil
}
func (s *stubAdapter) EstimateFee(_ context.Context, _ domain.TransferRequest) (domain.FeeEstimate, error) {
	return domain.FeeEstimate{}, nil
}
func (s *stubAdapter) BuildTransfer(_ context.Context, _ domain.TransferRequest) (domain.UnsignedTx, error) {
	return domain.UnsignedTx{}, nil
}
func (s *stubAdapter) Sign(_ context.Context, _ domain.UnsignedTx, _ port.Signer) (domain.SignedTx, error) {
	return domain.SignedTx{}, nil
}
func (s *stubAdapter) Broadcast(_ context.Context, _ domain.SignedTx) (string, error) { return "", nil }
func (s *stubAdapter) GetTransaction(_ context.Context, _ string) (domain.Transaction, error) {
	return domain.Transaction{}, nil
}

func TestRegistry_Adapter_Found(t *testing.T) {
	r := chainreg.NewRegistry([]port.ChainAdapter{
		&stubAdapter{domain.ChainEthereum},
		&stubAdapter{domain.ChainSolana},
	})

	a, err := r.Adapter(domain.ChainEthereum)
	if err != nil {
		t.Fatalf("Adapter: %v", err)
	}
	if a.Chain() != domain.ChainEthereum {
		t.Errorf("chain: got %s, want %s", a.Chain(), domain.ChainEthereum)
	}
}

func TestRegistry_Adapter_NotFound(t *testing.T) {
	r := chainreg.NewRegistry([]port.ChainAdapter{&stubAdapter{domain.ChainEthereum}})
	_, err := r.Adapter(domain.ChainBitcoin)
	if err == nil {
		t.Error("expected error for unregistered chain")
	}
}

func TestRegistry_Watcher_NotFound(t *testing.T) {
	r := chainreg.NewRegistry(nil)
	_, err := r.Watcher(domain.ChainEthereum)
	if err == nil {
		t.Error("expected error: no watchers registered")
	}
}

func TestRegistry_Supported(t *testing.T) {
	r := chainreg.NewRegistry([]port.ChainAdapter{
		&stubAdapter{domain.ChainEthereum},
		&stubAdapter{domain.ChainBitcoin},
		&stubAdapter{domain.ChainSolana},
	})

	supported := r.Supported()
	if len(supported) != 3 {
		t.Errorf("Supported: got %d chains, want 3", len(supported))
	}

	seen := make(map[domain.ChainID]bool, len(supported))
	for _, c := range supported {
		seen[c] = true
	}
	for _, c := range []domain.ChainID{domain.ChainEthereum, domain.ChainBitcoin, domain.ChainSolana} {
		if !seen[c] {
			t.Errorf("Supported: missing %s", c)
		}
	}
}

func TestRegistry_Empty(t *testing.T) {
	r := chainreg.NewRegistry(nil)
	if len(r.Supported()) != 0 {
		t.Error("empty registry should have no supported chains")
	}
}
