package port

import "github.com/karolchmiel94/omnicatena/internal/domain"

// Signer brokers signing without exposing private keys to the application layer:
// implementations resolve the key from an unlocked seed and sign within a
// controlled scope (ADR-0004).
type Signer interface {
	Sign(account domain.Account, digest []byte) ([]byte, error)
	PublicKey(account domain.Account) ([]byte, error)
}

// KeyStore persists wallet seeds encrypted at rest (Argon2id + AES-256-GCM).
// Plaintext seeds never leave the store except transiently for derivation and
// signing, and are never logged.
type KeyStore interface {
	// Create returns the freshly generated seed (plaintext, transient) for immediate derivation.
	Create(walletID string, passphrase []byte) (seed []byte, err error)
	Unlock(walletID string, passphrase []byte) (seed []byte, err error)
	Signer(walletID string, passphrase []byte) (Signer, error)
}
