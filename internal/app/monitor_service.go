package app

import (
	"context"

	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/port"
)

type MonitorService struct {
	registry  port.Registry
	publisher port.TxEventPublisher
}

func NewMonitorService(r port.Registry, p port.TxEventPublisher) *MonitorService {
	return &MonitorService{registry: r, publisher: p}
}

func (s *MonitorService) Watch(ctx context.Context, chain domain.ChainID, addrs []domain.Address) error {
	watcher, err := s.registry.Watcher(chain)
	if err != nil {
		return err
	}
	events, err := watcher.Watch(ctx, addrs)
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
			if err := s.publisher.Publish(ctx, evt); err != nil {
				return err
			}
		}
	}
}
