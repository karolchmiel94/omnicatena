package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/karolchmiel94/omnicatena/internal/domain"
	kafkago "github.com/segmentio/kafka-go"
)

// Topic auto-creation is asynchronous on the broker: a write racing right
// behind the topic's creation can still see UnknownTopicOrPartition. kafka-go's
// own tests retry on this for the same reason — see writer_test.go.
const autoCreateRetries = 5
const autoCreateRetryDelay = 250 * time.Millisecond

// Publisher implements port.TxEventPublisher over Kafka (ADR-0005).
type Publisher struct {
	writer *kafkago.Writer
}

func New(brokers []string, topic string) *Publisher {
	return &Publisher{
		writer: &kafkago.Writer{
			Addr:                   kafkago.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafkago.LeastBytes{},
			AllowAutoTopicCreation: true, // matches KAFKA_AUTO_CREATE_TOPICS_ENABLE on the local broker
		},
	}
}

func (p *Publisher) Publish(ctx context.Context, evt domain.TxEvent) error {
	msg, err := Encode(evt)
	if err != nil {
		return err
	}

	for attempt := 1; ; attempt++ {
		err = p.writer.WriteMessages(ctx, msg)
		if err == nil {
			return nil
		}
		if attempt >= autoCreateRetries || !errors.Is(err, kafkago.UnknownTopicOrPartition) {
			return fmt.Errorf("kafka: publish %s: %w", evt.Hash, err)
		}
		select {
		case <-time.After(autoCreateRetryDelay):
		case <-ctx.Done():
			return fmt.Errorf("kafka: publish %s: %w", evt.Hash, ctx.Err())
		}
	}
}

func (p *Publisher) Close() error {
	return p.writer.Close()
}

// Encode is pure (no I/O) so message construction is unit-testable without a
// broker. Keyed by tx hash so a chain's events for one tx land on one partition.
func Encode(evt domain.TxEvent) (kafkago.Message, error) {
	value, err := json.Marshal(evt)
	if err != nil {
		return kafkago.Message{}, fmt.Errorf("kafka: marshal event %s: %w", evt.Hash, err)
	}
	return kafkago.Message{Key: []byte(evt.Hash), Value: value}, nil
}
