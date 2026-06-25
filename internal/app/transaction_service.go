package app

import (
	"context"
	"fmt"

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

// Transfer runs the full native-transfer lifecycle: build → sign → broadcast.
func (s *TransactionService) Transfer(ctx context.Context, walletID string, passphrase []byte, req domain.TransferRequest) (string, error) {
	adapter, err := s.registry.Adapter(req.Network.Chain)
	if err != nil {
		return "", err
	}

	wallet, err := s.repo.Get(ctx, walletID)
	if err != nil {
		return "", err
	}
	account, ok := wallet.Account(req.Network.Chain)
	if !ok {
		return "", fmt.Errorf("transaction: no account for chain %s", req.Network.Chain)
	}

	signer, err := s.keys.Signer(walletID, passphrase)
	if err != nil {
		return "", err
	}

	unsigned, err := adapter.BuildTransfer(ctx, req)
	if err != nil {
		return "", err
	}
	unsigned.Account = account

	signed, err := adapter.Sign(ctx, unsigned, signer)
	if err != nil {
		return "", err
	}

	return adapter.Broadcast(ctx, signed)
}

func (s *TransactionService) Status(ctx context.Context, chain domain.ChainID, hash string) (domain.Transaction, error) {
	adapter, err := s.registry.Adapter(chain)
	if err != nil {
		return domain.Transaction{}, err
	}
	return adapter.GetTransaction(ctx, hash)
}
