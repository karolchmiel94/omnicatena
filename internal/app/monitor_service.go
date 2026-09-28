package app

import (
	"context"

	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/port"
)

type MonitorService struct {
	registry  port.Registry
	publisher port.TxEventPublisher
	txRepo    port.TransactionRepository
}

func NewMonitorService(r port.Registry, p port.TxEventPublisher, txRepo port.TransactionRepository) *MonitorService {
	return &MonitorService{registry: r, publisher: p, txRepo: txRepo}
}

// Watch scans chain for transactions touching any address in addrs (keyed by
// the wallet ID that owns it), persisting and publishing a TxEvent for each
// match. addrs may be empty — the watcher then just matches nothing.
func (s *MonitorService) Watch(ctx context.Context, chain domain.ChainID, addrs map[domain.Address]string) error {
	watcher, err := s.registry.Watcher(chain)
	if err != nil {
		return err
	}

	addrList := make([]domain.Address, 0, len(addrs))
	for addr := range addrs {
		addrList = append(addrList, addr)
	}

	events, err := watcher.Watch(ctx, addrList)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case evt, ok := <-events:
			if !ok {
				return nil
			}
			if err := s.txRepo.SaveEvent(ctx, addrs[evt.Address], evt); err != nil {
				return err
			}
			if err := s.publisher.Publish(ctx, evt); err != nil {
				return err
			}
		}
	}
}
