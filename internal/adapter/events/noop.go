// Package events holds port.TxEventPublisher implementations that aren't
// chain- or broker-specific (see the kafka subpackage for the Kafka one).
package events

import (
	"context"
	"log"

	"github.com/karolchmiel94/omnicatena/internal/domain"
)

// Noop discards events instead of publishing them — wired in when Kafka
// publishing is disabled (KAFKA_ENABLED=false) so ChainWatcher/MonitorService
// need no special-casing for "publishing is optional".
type Noop struct{}

func (Noop) Publish(_ context.Context, evt domain.TxEvent) error {
	log.Printf("event (publish disabled): chain=%s type=%s addr=%s hash=%s", evt.Chain, evt.Type, evt.Address, evt.Hash)
	return nil
}
