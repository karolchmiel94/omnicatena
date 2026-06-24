# ADR-0004: HD wallets with encrypted-at-rest seeds

- **Status:** Accepted
- **Date:** 2026-06-24

## Context

The platform is custodial for a single local user. We need to manage keys for
five chains across two elliptic-curve families. Options considered: one seed per
wallet deriving all chains, vs. independent per-chain keys.

## Decision

One **BIP-39 mnemonic per wallet**, deriving all chains hierarchically:
- **secp256k1** via BIP-32/44 for Bitcoin, Ethereum, Base, TRON.
- **ed25519** via SLIP-0010 for Solana.

The seed is **encrypted at rest** with Argon2id (KDF) + AES-256-GCM in a local
keystore. Plaintext seeds/keys never cross the application boundary: signing is
brokered by the `Signer` port over a transiently unlocked seed, and key material
is never logged (NFR3).

## Consequences

- **+** One wallet = one backup phrase = addresses on all five chains; realistic
  custodial UX and good portfolio material.
- **+** Curve differences are isolated inside each chain adapter's
  `DeriveAccount`.
- **−** A single seed compromise exposes all chains for that wallet — acceptable
  for a local prototype; **not** a production custody design (HSM/MPC out of
  scope, see non-goals).
- This is revisable: switching to per-chain keys would only change `KeyStore`
  and derivation, not the domain.
