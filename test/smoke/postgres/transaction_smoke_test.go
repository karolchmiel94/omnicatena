//go:build smoke

package smoke_test

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/karolchmiel94/omnicatena/internal/adapter/repository"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

// TestSmokeTransactionRepository: real Postgres — save a matched TxEvent,
// verify both the transactions row (from/to/amount/block_height) and the
// tx_addresses link row directly via SQL, then re-save with a changed status
// to prove the ON CONFLICT (chain, hash) upsert path works.
func TestSmokeTransactionRepository(t *testing.T) {
	pool := mustPool(t)
	ctx := context.Background()

	// tx_addresses.wallet_id has a FK to wallets — needs a real wallet row.
	walletRepo := repository.NewPostgresWallet(pool)
	suffix := time.Now().UnixNano()
	walletID := fmt.Sprintf("smoke-txw%d", suffix)
	if err := walletRepo.Save(ctx, domain.Wallet{ID: walletID, Label: "tx smoke", CreatedAt: time.Now()}); err != nil {
		t.Fatalf("Save wallet: %v", err)
	}

	txRepo := repository.NewPostgresTransaction(pool)
	hash := fmt.Sprintf("0xsmoketx%d", suffix)
	evt := domain.TxEvent{
		Type:         domain.EventInbound,
		Chain:        domain.ChainEthereum,
		Address:      "0xWatched",
		Counterparty: "0xSender",
		Hash:         hash,
		Amount:       domain.Amount{Asset: domain.Asset{Symbol: "ETH", Decimals: 18, Native: true}, Base: big.NewInt(123456789)},
		Status:       domain.TxConfirmed,
		BlockHeight:  42,
		Timestamp:    time.Now(),
	}

	if err := txRepo.SaveEvent(ctx, walletID, evt); err != nil {
		t.Fatalf("SaveEvent: %v", err)
	}

	var (
		gotStatus      string
		gotFrom, gotTo *string
		gotAmount      string
		gotBlockHeight *int64
	)
	err := pool.QueryRow(ctx,
		`SELECT status, from_address, to_address, amount, block_height FROM transactions WHERE chain = $1 AND hash = $2`,
		string(evt.Chain), evt.Hash,
	).Scan(&gotStatus, &gotFrom, &gotTo, &gotAmount, &gotBlockHeight)
	if err != nil {
		t.Fatalf("verify transactions row: %v", err)
	}
	if gotStatus != "confirmed" {
		t.Errorf("status = %s, want confirmed", gotStatus)
	}
	if gotFrom == nil || *gotFrom != string(evt.Counterparty) {
		t.Errorf("from_address = %v, want %s", gotFrom, evt.Counterparty)
	}
	if gotTo == nil || *gotTo != string(evt.Address) {
		t.Errorf("to_address = %v, want %s", gotTo, evt.Address)
	}
	if gotAmount != "123456789" {
		t.Errorf("amount = %s, want 123456789", gotAmount)
	}
	if gotBlockHeight == nil || *gotBlockHeight != 42 {
		t.Errorf("block_height = %v, want 42", gotBlockHeight)
	}

	var role string
	err = pool.QueryRow(ctx, `SELECT role FROM tx_addresses WHERE wallet_id = $1 AND address = $2`, walletID, string(evt.Address)).Scan(&role)
	if err != nil {
		t.Fatalf("verify tx_addresses row: %v", err)
	}
	if role != "receiver" {
		t.Errorf("role = %s, want receiver", role)
	}

	// Re-observe the same tx with a different status — must update in place, not duplicate.
	evt.Status = domain.TxFailed
	if err := txRepo.SaveEvent(ctx, walletID, evt); err != nil {
		t.Fatalf("SaveEvent (update): %v", err)
	}
	var gotStatus2 string
	if err := pool.QueryRow(ctx, `SELECT status FROM transactions WHERE chain = $1 AND hash = $2`, string(evt.Chain), evt.Hash).Scan(&gotStatus2); err != nil {
		t.Fatalf("verify updated status: %v", err)
	}
	if gotStatus2 != "failed" {
		t.Errorf("after update: status = %s, want failed", gotStatus2)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM transactions WHERE chain = $1 AND hash = $2`, string(evt.Chain), evt.Hash).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Errorf("row count = %d, want 1 (update should not duplicate)", count)
	}
}
