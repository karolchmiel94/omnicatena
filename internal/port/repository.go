package port

import (
	"context"

	"github.com/karolchmiel94/omnicatena/internal/domain"
)

// WalletRepository persists wallet metadata (addresses, labels) — never secrets.
type WalletRepository interface {
	Save(ctx context.Context, w domain.Wallet) error
	Get(ctx context.Context, id string) (domain.Wallet, error)
	List(ctx context.Context) ([]domain.Wallet, error)
}

// TransactionRepository persists transactions a ChainWatcher observes
// involving a watched address (schema: transactions + tx_addresses).
// Chain-agnostic — chain is data (domain.ChainID), not part of the interface,
// so one implementation serves every chain.
type TransactionRepository interface {
	// SaveEvent upserts the transaction and links it to walletID (the wallet
	// that owns evt.Address). Idempotent — re-observing the same tx (e.g. once
	// pending, again once confirmed) updates the existing row.
	SaveEvent(ctx context.Context, walletID string, evt domain.TxEvent) error
}

// ChainCursorStore persists the last block height a ChainWatcher scanned, so
// it can resume on restart instead of rescanning from the chain tip. Optional
// for a watcher to use — without one, Watch just starts from the current tip,
// as before persistence existed.
type ChainCursorStore interface {
	Get(ctx context.Context, chain domain.ChainID) (height uint64, ok bool, err error)
	Set(ctx context.Context, chain domain.ChainID, height uint64) error
}
