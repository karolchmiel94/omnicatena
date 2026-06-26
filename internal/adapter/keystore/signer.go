package keystore

import (
	"crypto/ecdsa"
	"crypto/ed25519"

	btcec "github.com/btcsuite/btcd/btcec/v2"
	btcecdsa "github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/hdwallet"
)

type memSigner struct {
	seed []byte
}

// Sign returns the signature over digest for the given account's chain:
//   - Bitcoin: DER-encoded secp256k1 (without SIGHASH_ALL — adapter appends it)
//   - EVM/TRON/Base: [R || S || V] (65 bytes, go-ethereum convention)
//   - Solana: raw 64-byte ed25519 signature over the message (not a hash)
func (s *memSigner) Sign(account domain.Account, digest []byte) ([]byte, error) {
	if account.Chain == domain.ChainSolana {
		priv, err := deriveEd25519Key(s.seed, account.Path)
		if err != nil {
			return nil, err
		}
		return ed25519.Sign(priv, digest), nil
	}
	priv, err := derivePrivKey(s.seed, account.Path)
	if err != nil {
		return nil, err
	}
	if account.Chain == domain.ChainBitcoin {
		btcKey, _ := btcec.PrivKeyFromBytes(priv.D.FillBytes(make([]byte, 32)))
		return btcecdsa.Sign(btcKey, digest).Serialize(), nil
	}
	return crypto.Sign(digest, priv)
}

// PublicKey returns the compressed 33-byte secp256k1 public key, or the 32-byte
// ed25519 public key for Solana.
func (s *memSigner) PublicKey(account domain.Account) ([]byte, error) {
	if account.Chain == domain.ChainSolana {
		priv, err := deriveEd25519Key(s.seed, account.Path)
		if err != nil {
			return nil, err
		}
		return []byte(priv.Public().(ed25519.PublicKey)), nil
	}
	priv, err := derivePrivKey(s.seed, account.Path)
	if err != nil {
		return nil, err
	}
	btcKey, _ := btcec.PrivKeyFromBytes(priv.D.FillBytes(make([]byte, 32)))
	return btcKey.PubKey().SerializeCompressed(), nil
}

func deriveEd25519Key(seed []byte, path domain.DerivationPath) (ed25519.PrivateKey, error) {
	keySeed, err := hdwallet.DeriveKeyEd25519(seed, path)
	if err != nil {
		return nil, err
	}
	return ed25519.NewKeyFromSeed(keySeed), nil
}

func derivePrivKey(seed []byte, path domain.DerivationPath) (*ecdsa.PrivateKey, error) {
	key, err := hdwallet.DeriveKey(seed, path)
	if err != nil {
		return nil, err
	}
	privKey, err := key.ECPrivKey()
	if err != nil {
		return nil, err
	}
	return privKey.ToECDSA(), nil
}
