//go:build smoke

package smoke_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/karolchmiel94/omnicatena/internal/adapter/repository"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

const postgresDSN = "postgres://omni:omni@localhost:5433/omnicatena?sslmode=disable"

func mustPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), postgresDSN)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// TestSmokeWalletRepository: real Postgres — save a wallet with accounts on
// two chains, read it back, confirm it shows up in List, then re-save with a
// changed address to prove the ON CONFLICT (upsert) path works too.
func TestSmokeWalletRepository(t *testing.T) {
	pool := mustPool(t)
	repo := repository.NewPostgresWallet(pool)
	ctx := context.Background()

	suffix := time.Now().UnixNano()
	id := fmt.Sprintf("smoke-w%d", suffix)
	w := domain.Wallet{
		ID:        id,
		Label:     "smoke test wallet",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
		Accounts: []domain.Account{
			{Chain: domain.ChainEthereum, Address: domain.Address(fmt.Sprintf("0xSmokeEth%d", suffix)), Path: "m/44'/60'/0'/0/0"},
			{Chain: domain.ChainBitcoin, Address: domain.Address(fmt.Sprintf("mSmokeBtc%d", suffix)), Path: "m/44'/0'/0'/0/0"},
		},
	}

	if err := repo.Save(ctx, w); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Label != w.Label {
		t.Errorf("label = %q, want %q", got.Label, w.Label)
	}
	if len(got.Accounts) != 2 {
		t.Fatalf("accounts = %d, want 2: %+v", len(got.Accounts), got.Accounts)
	}
	ethAcc, ok := got.Account(domain.ChainEthereum)
	if !ok || ethAcc.Address != w.Accounts[0].Address {
		t.Errorf("eth account = %+v, want address %s", ethAcc, w.Accounts[0].Address)
	}

	all, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, ww := range all {
		if ww.ID == id {
			found = true
		}
	}
	if !found {
		t.Error("saved wallet not found in List")
	}

	// Re-save with an updated account address — confirms the ON CONFLICT path.
	updated := domain.Address(fmt.Sprintf("0xSmokeEthUpdated%d", suffix))
	w.Accounts[0].Address = updated
	if err := repo.Save(ctx, w); err != nil {
		t.Fatalf("Save (update): %v", err)
	}
	got2, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get (after update): %v", err)
	}
	ethAcc2, _ := got2.Account(domain.ChainEthereum)
	if ethAcc2.Address != updated {
		t.Errorf("after update: eth address = %s, want %s", ethAcc2.Address, updated)
	}
}
