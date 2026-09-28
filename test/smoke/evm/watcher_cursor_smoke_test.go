//go:build smoke

package smoke_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/karolchmiel94/omnicatena/internal/adapter/chain/evm"
	"github.com/karolchmiel94/omnicatena/internal/adapter/repository"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

const postgresDSN = "postgres://omni:omni@localhost:5433/omnicatena?sslmode=disable"

// TestSmokeWatcherCursorPersistence: real Anvil + real Postgres — a watcher
// wired with SetCursorStore actually writes its scan progress to
// chain_cursors as it processes blocks (not just that the store works in
// isolation, which test/smoke/postgres covers separately).
func TestSmokeWatcherCursorPersistence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, postgresDSN)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	defer pool.Close()
	cursors := repository.NewPostgresCursor(pool)

	// Isolated chain id so this doesn't collide with a real chain's cursor row.
	chain := domain.ChainID("smoke-cursor-evm")

	watcher, err := evm.NewWatcher(evm.Config{RPCURL: anvilRPCURL, ChainID: anvilChainID, Chain: chain})
	if err != nil {
		t.Fatalf("evm.NewWatcher: %v", err)
	}
	watcher.SetCursorStore(cursors)

	events, err := watcher.Watch(ctx, nil) // no watched addresses — only need blocks to advance
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}
	go func() {
		for range events {
		}
	}()

	rawClient, err := ethclient.Dial(anvilRPCURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	toKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	toAddr := crypto.PubkeyToAddress(toKey.PublicKey)
	fundAddress(ctx, t, rawClient, toAddr, big.NewInt(1)) // mines a block

	var height uint64
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
poll:
	for {
		select {
		case <-tick.C:
			h, ok, err := cursors.Get(ctx, chain)
			if err != nil {
				t.Fatalf("Get cursor: %v", err)
			}
			if ok && h > 0 {
				height = h
				break poll
			}
		case <-ctx.Done():
			t.Fatal("timed out waiting for cursor to advance")
		}
	}
	t.Logf("cursor persisted at block %d", height)
}
