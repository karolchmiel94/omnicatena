package unit_test

import (
	"context"
	"testing"

	"github.com/karolchmiel94/omnicatena/internal/adapter/events"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

func TestNoopPublisher_NeverErrors(t *testing.T) {
	var pub events.Noop
	err := pub.Publish(context.Background(), domain.TxEvent{Hash: "0x1"})
	if err != nil {
		t.Errorf("Publish: got %v, want nil", err)
	}
}
