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
	Type      TxEventType
	Chain     ChainID
	Address   Address
	Hash      string
	Amount    Amount
	Status    TxStatus
	Timestamp time.Time
}
