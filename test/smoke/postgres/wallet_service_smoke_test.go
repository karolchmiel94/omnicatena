//go:build smoke

package smoke_test

import (
	"context"
	"testing"

	chainreg "github.com/karolchmiel94/omnicatena/internal/adapter/chain"
	"github.com/karolchmiel94/omnicatena/internal/adapter/keystore"
	"github.com/karolchmiel94/omnicatena/internal/adapter/repository"
	"github.com/karolchmiel94/omnicatena/internal/app"
)

// TestSmokeWalletServiceCreate: real Postgres — exercises app.WalletService.Create
// exactly as cmd/api and cmd/cli wire it (Postgres-backed WalletRepository +
// KeyStore). This is what catches the keystore.wallet_id foreign key ordering
// bug: Create must persist the wallet row before writing to keystore, not
// after, or the keystore insert fails against a real FK constraint (the
// in-memory adapters never enforce it, so this only surfaces with Postgres).
func TestSmokeWalletServiceCreate(t *testing.T) {
	pool := mustPool(t)
	ctx := context.Background()

	registry := chainreg.NewRegistry(nil) // no chain adapters needed to exercise the persistence path
	keys := keystore.NewPostgres(pool)
	repo := repository.NewPostgresWallet(pool)
	svc := app.NewWalletService(registry, keys, repo)

	w, err := svc.Create(ctx, "smoke wallet service", []byte("passphrase"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if w.ID == "" {
		t.Fatal("Create returned empty wallet ID")
	}

	got, err := repo.Get(ctx, w.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Label != w.Label {
		t.Errorf("label = %q, want %q", got.Label, w.Label)
	}

	// The point of the fix: the keystore row must exist too, and be unlockable.
	signer, err := keys.Signer(w.ID, []byte("passphrase"))
	if err != nil {
		t.Fatalf("Signer: %v", err)
	}
	if signer == nil {
		t.Fatal("Signer returned nil")
	}
}
