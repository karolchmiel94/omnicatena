package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/karolchmiel94/omnicatena/internal/adapter/repository"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

func TestInMemoryWallet_SaveGet(t *testing.T) {
	r := repository.NewInMemoryWallet()
	ctx := context.Background()

	w := domain.Wallet{ID: "w1", Label: "test", CreatedAt: time.Now()}
	if err := r.Save(ctx, w); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := r.Get(ctx, "w1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != w.ID || got.Label != w.Label {
		t.Errorf("got {%s %s}, want {%s %s}", got.ID, got.Label, w.ID, w.Label)
	}
}

func TestInMemoryWallet_Get_NotFound(t *testing.T) {
	r := repository.NewInMemoryWallet()
	_, err := r.Get(context.Background(), "missing")
	if err == nil {
		t.Error("expected error for missing wallet ID")
	}
}

func TestInMemoryWallet_List(t *testing.T) {
	r := repository.NewInMemoryWallet()
	ctx := context.Background()

	for _, w := range []domain.Wallet{
		{ID: "w1", Label: "a"},
		{ID: "w2", Label: "b"},
		{ID: "w3", Label: "c"},
	} {
		if err := r.Save(ctx, w); err != nil {
			t.Fatal(err)
		}
	}

	list, err := r.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("List: got %d wallets, want 3", len(list))
	}
}

func TestInMemoryWallet_List_Empty(t *testing.T) {
	r := repository.NewInMemoryWallet()
	list, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("empty List: got %d wallets, want 0", len(list))
	}
}

func TestInMemoryWallet_Save_Overwrites(t *testing.T) {
	r := repository.NewInMemoryWallet()
	ctx := context.Background()

	_ = r.Save(ctx, domain.Wallet{ID: "w1", Label: "original"})
	_ = r.Save(ctx, domain.Wallet{ID: "w1", Label: "updated"})

	got, _ := r.Get(ctx, "w1")
	if got.Label != "updated" {
		t.Errorf("expected updated label, got %q", got.Label)
	}

	list, _ := r.List(ctx)
	if len(list) != 1 {
		t.Errorf("expected 1 wallet after overwrite, got %d", len(list))
	}
}
