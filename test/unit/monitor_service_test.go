package unit_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	chainreg "github.com/karolchmiel94/omnicatena/internal/adapter/chain"
	"github.com/karolchmiel94/omnicatena/internal/app"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

// scriptedWatcher emits a fixed sequence of events then closes the channel —
// no goroutine/RPC involved, so Watch's persist+publish loop is testable
// synchronously.
type scriptedWatcher struct {
	chain  domain.ChainID
	events []domain.TxEvent
}

func (w *scriptedWatcher) Chain() domain.ChainID { return w.chain }
func (w *scriptedWatcher) Watch(_ context.Context, _ []domain.Address) (<-chan domain.TxEvent, error) {
	out := make(chan domain.TxEvent, len(w.events))
	for _, e := range w.events {
		out <- e
	}
	close(out)
	return out, nil
}

type recordingPublisher struct {
	mu   sync.Mutex
	seen []domain.TxEvent
}

func (p *recordingPublisher) Publish(_ context.Context, evt domain.TxEvent) error {
	p.mu.Lock()
	p.seen = append(p.seen, evt)
	p.mu.Unlock()
	return nil
}

type saveCall struct {
	walletID string
	evt      domain.TxEvent
}

type recordingTxRepo struct {
	mu    sync.Mutex
	saves []saveCall
}

func (r *recordingTxRepo) SaveEvent(_ context.Context, walletID string, evt domain.TxEvent) error {
	r.mu.Lock()
	r.saves = append(r.saves, saveCall{walletID, evt})
	r.mu.Unlock()
	return nil
}

func TestMonitorService_Watch_PersistsAndPublishes(t *testing.T) {
	evtA := domain.TxEvent{Chain: domain.ChainEthereum, Address: "0xA", Hash: "0x1", Type: domain.EventInbound}
	evtB := domain.TxEvent{Chain: domain.ChainEthereum, Address: "0xB", Hash: "0x2", Type: domain.EventOutbound}

	watcher := &scriptedWatcher{chain: domain.ChainEthereum, events: []domain.TxEvent{evtA, evtB}}
	registry := chainreg.NewRegistry(nil)
	registry.RegisterWatcher(watcher)

	pub := &recordingPublisher{}
	txRepo := &recordingTxRepo{}
	svc := app.NewMonitorService(registry, pub, txRepo)

	addrs := map[domain.Address]string{"0xA": "wallet-1", "0xB": "wallet-2"}
	if err := svc.Watch(context.Background(), domain.ChainEthereum, addrs); err != nil {
		t.Fatalf("Watch: %v", err)
	}

	if len(pub.seen) != 2 {
		t.Fatalf("published %d events, want 2", len(pub.seen))
	}
	if len(txRepo.saves) != 2 {
		t.Fatalf("saved %d events, want 2", len(txRepo.saves))
	}

	byHash := make(map[string]saveCall)
	for _, s := range txRepo.saves {
		byHash[s.evt.Hash] = s
	}
	if byHash["0x1"].walletID != "wallet-1" {
		t.Errorf("event 0x1: walletID = %q, want wallet-1", byHash["0x1"].walletID)
	}
	if byHash["0x2"].walletID != "wallet-2" {
		t.Errorf("event 0x2: walletID = %q, want wallet-2", byHash["0x2"].walletID)
	}
}

func TestMonitorService_Watch_UnknownChain(t *testing.T) {
	registry := chainreg.NewRegistry(nil)
	svc := app.NewMonitorService(registry, &recordingPublisher{}, &recordingTxRepo{})

	if err := svc.Watch(context.Background(), domain.ChainBitcoin, nil); err == nil {
		t.Error("expected error for unregistered chain")
	}
}

type erroringTxRepo struct{}

func (erroringTxRepo) SaveEvent(context.Context, string, domain.TxEvent) error {
	return errors.New("boom")
}

func TestMonitorService_Watch_StopsOnSaveError(t *testing.T) {
	watcher := &scriptedWatcher{chain: domain.ChainEthereum, events: []domain.TxEvent{{Hash: "0x1"}}}
	registry := chainreg.NewRegistry(nil)
	registry.RegisterWatcher(watcher)

	svc := app.NewMonitorService(registry, &recordingPublisher{}, erroringTxRepo{})
	if err := svc.Watch(context.Background(), domain.ChainEthereum, nil); err == nil {
		t.Error("expected error propagated from SaveEvent")
	}
}
