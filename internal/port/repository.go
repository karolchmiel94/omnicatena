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
