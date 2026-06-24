package app

import (
	"context"
	"errors"

	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/port"
)

var errNotImplemented = errors.New("omnicatena: not implemented")

// WalletService is delivery-agnostic: the HTTP API and CLI both call it.
type WalletService struct {
	registry port.Registry
	keys     port.KeyStore
	repo     port.WalletRepository
}

func NewWalletService(r port.Registry, k port.KeyStore, repo port.WalletRepository) *WalletService {
	return &WalletService{registry: r, keys: k, repo: repo}
}

// Create generates+encrypts a seed, derives one account per supported chain, and persists the wallet.
func (s *WalletService) Create(ctx context.Context, label string, passphrase []byte) (domain.Wallet, error) {
	return domain.Wallet{}, errNotImplemented
}

func (s *WalletService) Get(ctx context.Context, id string) (domain.Wallet, error) {
	return s.repo.Get(ctx, id)
}

func (s *WalletService) List(ctx context.Context) ([]domain.Wallet, error) {
	return s.repo.List(ctx)
}

func (s *WalletService) Balance(ctx context.Context, walletID string, chain domain.ChainID) (domain.Amount, error) {
	return domain.Amount{}, errNotImplemented
}
