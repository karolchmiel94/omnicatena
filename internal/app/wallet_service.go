package app

import (
	"context"
	"fmt"
	"time"

	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/port"
)

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
	id := fmt.Sprintf("w%d", time.Now().UnixNano())

	seed, err := s.keys.Create(id, passphrase)
	if err != nil {
		return domain.Wallet{}, fmt.Errorf("wallet: create keys: %w", err)
	}

	w := domain.Wallet{
		ID:        id,
		Label:     label,
		CreatedAt: time.Now(),
	}

	for _, chainID := range s.registry.Supported() {
		adapter, err := s.registry.Adapter(chainID)
		if err != nil {
			continue
		}
		account, err := adapter.DeriveAccount(seed, bip44Path(chainID))
		if err != nil {
			continue
		}
		w.Accounts = append(w.Accounts, account)
	}

	if err := s.repo.Save(ctx, w); err != nil {
		return domain.Wallet{}, fmt.Errorf("wallet: save: %w", err)
	}
	return w, nil
}

func (s *WalletService) Get(ctx context.Context, id string) (domain.Wallet, error) {
	return s.repo.Get(ctx, id)
}

func (s *WalletService) List(ctx context.Context) ([]domain.Wallet, error) {
	return s.repo.List(ctx)
}

func (s *WalletService) Balance(ctx context.Context, walletID string, chain domain.ChainID) (domain.Amount, error) {
	wallet, err := s.repo.Get(ctx, walletID)
	if err != nil {
		return domain.Amount{}, err
	}
	account, ok := wallet.Account(chain)
	if !ok {
		return domain.Amount{}, fmt.Errorf("wallet: no account for chain %s", chain)
	}
	adapter, err := s.registry.Adapter(chain)
	if err != nil {
		return domain.Amount{}, err
	}
	return adapter.Balance(ctx, account.Address, domain.Asset{Native: true})
}

// bip44Path returns the standard BIP-44 path for the first account on a chain.
// Coin types: ETH/Base=60, BTC=0, SOL=501, TRON=195.
var bip44CoinType = map[domain.ChainID]uint32{
	domain.ChainEthereum: 60,
	domain.ChainBase:     60,
	domain.ChainBitcoin:  0,
	domain.ChainSolana:   501,
	domain.ChainTron:     195,
}

func bip44Path(chain domain.ChainID) domain.DerivationPath {
	coin := bip44CoinType[chain]
	return domain.DerivationPath(fmt.Sprintf("m/44'/%d'/0'/0/0", coin))
}
