package app

import (
	"context"

	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/port"
)

type TransactionService struct {
	registry port.Registry
	keys     port.KeyStore
	repo     port.WalletRepository
}

func NewTransactionService(r port.Registry, k port.KeyStore, repo port.WalletRepository) *TransactionService {
	return &TransactionService{registry: r, keys: k, repo: repo}
}

func (s *TransactionService) EstimateFee(ctx context.Context, req domain.TransferRequest) (domain.FeeEstimate, error) {
	adapter, err := s.registry.Adapter(req.Network.Chain)
	if err != nil {
		return domain.FeeEstimate{}, err
	}
	return adapter.EstimateFee(ctx, req)
}

// Transfer runs the full native-transfer lifecycle: estimate → build → sign → broadcast.
func (s *TransactionService) Transfer(ctx context.Context, walletID string, passphrase []byte, req domain.TransferRequest) (string, error) {
	return "", errNotImplemented
}

func (s *TransactionService) Status(ctx context.Context, chain domain.ChainID, hash string) (domain.Transaction, error) {
	adapter, err := s.registry.Adapter(chain)
	if err != nil {
		return domain.Transaction{}, err
	}
	return adapter.GetTransaction(ctx, hash)
}
