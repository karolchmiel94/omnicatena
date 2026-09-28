//go:build smoke

package smoke_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/karolchmiel94/omnicatena/internal/adapter/repository"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

// TestSmokeChainCursorStore: real Postgres — a cursor that was never set
// reports ok=false, Set then Get round-trips the height, and a second Set
// updates in place rather than erroring.
func TestSmokeChainCursorStore(t *testing.T) {
	pool := mustPool(t)
	ctx := context.Background()
	store := repository.NewPostgresCursor(pool)

	chain := domain.ChainID(fmt.Sprintf("smoke-chain-%d", time.Now().UnixNano()))

	if _, ok, err := store.Get(ctx, chain); err != nil {
		t.Fatalf("Get (missing): %v", err)
	} else if ok {
		t.Error("expected ok=false for a cursor that was never set")
	}

	if err := store.Set(ctx, chain, 100); err != nil {
		t.Fatalf("Set: %v", err)
	}
	height, ok, err := store.Get(ctx, chain)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok || height != 100 {
		t.Errorf("got height=%d ok=%v, want 100/true", height, ok)
	}

	if err := store.Set(ctx, chain, 250); err != nil {
		t.Fatalf("Set (update): %v", err)
	}
	height2, _, err := store.Get(ctx, chain)
	if err != nil {
		t.Fatalf("Get (after update): %v", err)
	}
	if height2 != 250 {
		t.Errorf("after update: height = %d, want 250", height2)
	}
}
