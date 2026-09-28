//go:build smoke

package smoke_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/karolchmiel94/omnicatena/internal/adapter/keystore"
	"github.com/karolchmiel94/omnicatena/internal/adapter/repository"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

// TestSmokeKeyStorePostgres: real Postgres — create seals a seed and persists
// it, Unlock with the right passphrase returns the same seed, the wrong
// passphrase fails, and Signer works off the unlocked seed.
func TestSmokeKeyStorePostgres(t *testing.T) {
	pool := mustPool(t)
	ks := keystore.NewPostgres(pool)

	// keystore.wallet_id has a FK to wallets(id) — needs a real wallet row.
	walletID := fmt.Sprintf("smoke-ks%d", time.Now().UnixNano())
	walletRepo := repository.NewPostgresWallet(pool)
	if err := walletRepo.Save(context.Background(), domain.Wallet{ID: walletID, Label: "keystore smoke", CreatedAt: time.Now()}); err != nil {
		t.Fatalf("Save wallet: %v", err)
	}

	passphrase := []byte("correct horse battery staple")

	seed, err := ks.Create(walletID, passphrase)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(seed) == 0 {
		t.Fatal("Create returned empty seed")
	}

	unlocked, err := ks.Unlock(walletID, passphrase)
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if string(unlocked) != string(seed) {
		t.Error("unlocked seed does not match created seed")
	}

	if _, err := ks.Unlock(walletID, []byte("wrong passphrase")); err == nil {
		t.Error("expected error unlocking with wrong passphrase")
	}

	signer, err := ks.Signer(walletID, passphrase)
	if err != nil {
		t.Fatalf("Signer: %v", err)
	}
	if signer == nil {
		t.Fatal("Signer returned nil")
	}
}
