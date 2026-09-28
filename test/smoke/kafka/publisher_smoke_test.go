//go:build smoke

package smoke_test

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/karolchmiel94/omnicatena/internal/adapter/events/kafka"
	"github.com/karolchmiel94/omnicatena/internal/domain"
	kafkago "github.com/segmentio/kafka-go"
)

const kafkaBrokerAddr = "localhost:9092"

// TestSmokePublisher: publish a real TxEvent to the local Kafka broker, then
// read it back with a plain consumer to confirm it actually landed on the
// topic, not just that WriteMessages returned nil.
func TestSmokePublisher(t *testing.T) {
	topic := "omnicatena.tx.events.smoke"
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pub := kafka.New([]string{kafkaBrokerAddr}, topic)
	defer pub.Close()

	want := domain.TxEvent{
		Type:      domain.EventInbound,
		Chain:     domain.ChainEthereum,
		Address:   domain.Address("0xsmoke"),
		Hash:      "0xsmoketest",
		Amount:    domain.Amount{Asset: domain.Asset{Symbol: "ETH", Decimals: 18, Native: true}, Base: big.NewInt(42)},
		Status:    domain.TxConfirmed,
		Timestamp: time.Now().UTC().Truncate(time.Second),
	}

	if err := pub.Publish(ctx, want); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:   []string{kafkaBrokerAddr},
		Topic:     topic,
		Partition: 0,
	})
	defer reader.Close()

	msg, err := reader.ReadMessage(ctx)
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}

	var got domain.TxEvent
	if err := json.Unmarshal(msg.Value, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Hash != want.Hash || got.Chain != want.Chain || got.Type != want.Type {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if got.Amount.Base.Cmp(want.Amount.Base) != 0 {
		t.Errorf("amount = %s, want %s", got.Amount.Base, want.Amount.Base)
	}
}
