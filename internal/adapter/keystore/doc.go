// Package keystore implements port.KeyStore and port.Signer: BIP-39 seeds stored
// encrypted at rest (Argon2id + AES-256-GCM), with signing scoped to a transiently
// unlocked seed (ADR-0004). Plaintext key material never crosses the application
// boundary and is never logged.
package keystore
