package port

import (
	"context"

	"github.com/karolchmiel94/omnicatena/internal/domain"
)

// Kafka implements this in V1; tests and local runs can substitute an in-memory
// or stdout publisher (ADR-0005).
type TxEventPublisher interface {
	Publish(ctx context.Context, evt domain.TxEvent) error
}
