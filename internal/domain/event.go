package domain

import "time"

type TxEventType string

const (
	EventInbound   TxEventType = "inbound"
	EventOutbound  TxEventType = "outbound"
	EventConfirmed TxEventType = "confirmed"
)

// Published to the outbound stream (Kafka in V1); the unit V2 traffic/cost monitoring will enrich.
type TxEvent struct {
	Type  TxEventType
	Chain ChainID
	// Address is the watched side; Counterparty is the other, when known (may
	// be empty — e.g. a contract-creation tx has no recipient).
	Address      Address
	Counterparty Address
	Hash         string
	Amount       Amount
	Status       TxStatus
	BlockHeight  uint64
	Timestamp    time.Time
}
