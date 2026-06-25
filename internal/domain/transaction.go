package domain

type TxStatus string

const (
	TxPending   TxStatus = "pending"
	TxConfirmed TxStatus = "confirmed"
	TxFailed    TxStatus = "failed"
	TxUnknown   TxStatus = "unknown"
)

type TransferRequest struct {
	Network Network
	From    Address
	To      Address
	Amount  Amount
	Speed   FeeSpeed
}

// UnsignedTx is the envelope pattern (ADR-0006): a uniform wrapper over a
// chain-specific, opaque Payload. The application layer never inspects Payload;
// only the adapter that produced it can sign it.
type UnsignedTx struct {
	Chain   ChainID
	Account Account // signing account — carries the derivation path the Signer needs
	Request TransferRequest
	Fee     FeeEstimate
	Payload []byte
}

type SignedTx struct {
	Chain ChainID
	Raw   []byte
	Hash  string
}

type Transaction struct {
	Chain         ChainID
	Hash          string
	Status        TxStatus
	Confirmations uint64
	BlockHeight   uint64
	Fee           Amount
}
