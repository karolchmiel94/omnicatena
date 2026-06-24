package port

import (
	"context"

	"github.com/karolchmiel94/omnicatena/internal/domain"
)

// ChainAdapter is the driven port every blockchain integration implements. One
// adapter serves one chain family — the EVM adapter backs both Ethereum and Base
// via config, not separate code (ADR-0007). Adding a chain means implementing
// this interface and registering it; the domain and use cases do not change.
type ChainAdapter interface {
	Chain() domain.ChainID

	// Curve and address encoding are chain-specific: secp256k1 (BIP-32/44) for
	// BTC/EVM/TRON, ed25519 (SLIP-0010) for Solana.
	DeriveAccount(seed []byte, path domain.DerivationPath) (domain.Account, error)

	Balance(ctx context.Context, addr domain.Address, asset domain.Asset) (domain.Amount, error)

	// EstimateFee may be naive in V1 (node-suggested); V2 improves it behind this
	// same signature (ADR-0008).
	EstimateFee(ctx context.Context, req domain.TransferRequest) (domain.FeeEstimate, error)

	BuildTransfer(ctx context.Context, req domain.TransferRequest) (domain.UnsignedTx, error)

	// Sign uses the Signer to broker key access without exposing private keys to the caller.
	Sign(ctx context.Context, tx domain.UnsignedTx, signer Signer) (domain.SignedTx, error)

	Broadcast(ctx context.Context, tx domain.SignedTx) (string, error)
	GetTransaction(ctx context.Context, hash string) (domain.Transaction, error)
}

// ChainWatcher is the monitoring seam. V1 may poll; V2 swaps in implementations
// that also sample fee markets and emit traffic/cost telemetry, without callers
// changing (ADR-0008).
type ChainWatcher interface {
	Chain() domain.ChainID
	// Watch streams events until ctx is cancelled.
	Watch(ctx context.Context, addrs []domain.Address) (<-chan domain.TxEvent, error)
}

// Registry is the single place that knows which chains are wired in.
type Registry interface {
	Adapter(chain domain.ChainID) (ChainAdapter, error)
	Watcher(chain domain.ChainID) (ChainWatcher, error)
	Supported() []domain.ChainID
}
