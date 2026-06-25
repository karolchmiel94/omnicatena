package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/karolchmiel94/omnicatena/internal/domain"
)

type InMemoryWallet struct {
	mu      sync.RWMutex
	wallets map[string]domain.Wallet
}

func NewInMemoryWallet() *InMemoryWallet {
	return &InMemoryWallet{wallets: make(map[string]domain.Wallet)}
}

func (r *InMemoryWallet) Save(_ context.Context, w domain.Wallet) error {
	r.mu.Lock()
	r.wallets[w.ID] = w
	r.mu.Unlock()
	return nil
}

func (r *InMemoryWallet) Get(_ context.Context, id string) (domain.Wallet, error) {
	r.mu.RLock()
	w, ok := r.wallets[id]
	r.mu.RUnlock()
	if !ok {
		return domain.Wallet{}, fmt.Errorf("repository: wallet %s not found", id)
	}
	return w, nil
}

func (r *InMemoryWallet) List(_ context.Context) ([]domain.Wallet, error) {
	r.mu.RLock()
	out := make([]domain.Wallet, 0, len(r.wallets))
	for _, w := range r.wallets {
		out = append(out, w)
	}
	r.mu.RUnlock()
	return out, nil
}
