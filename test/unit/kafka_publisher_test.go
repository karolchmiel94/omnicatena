package unit_test

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/karolchmiel94/omnicatena/internal/adapter/events/kafka"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

func TestKafkaEncode(t *testing.T) {
	evt := domain.TxEvent{
		Type:      domain.EventInbound,
		Chain:     domain.ChainEthereum,
		Address:   domain.Address("0xabc"),
		Hash:      "0xdeadbeef",
		Amount:    domain.Amount{Asset: domain.Asset{Symbol: "ETH", Decimals: 18, Native: true}, Base: big.NewInt(1)},
		Status:    domain.TxConfirmed,
		Timestamp: time.Unix(1_700_000_000, 0).UTC(),
	}

	msg, err := kafka.Encode(evt)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if string(msg.Key) != evt.Hash {
		t.Errorf("key = %q, want %q", msg.Key, evt.Hash)
	}

	var decoded domain.TxEvent
	if err := json.Unmarshal(msg.Value, &decoded); err != nil {
		t.Fatalf("unmarshal value: %v", err)
	}
	if decoded.Hash != evt.Hash || decoded.Chain != evt.Chain || decoded.Type != evt.Type {
		t.Errorf("decoded = %+v, want %+v", decoded, evt)
	}
}
