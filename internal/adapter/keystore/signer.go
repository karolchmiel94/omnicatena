package keystore

import (
	"crypto/ecdsa"

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
//   - Bitcoin: DER-encoded secp256k1 signature (without SIGHASH_ALL byte — adapter appends it)
//   - EVM/TRON/Base: [R || S || V] (65 bytes, go-ethereum convention)
func (s *memSigner) Sign(account domain.Account, digest []byte) ([]byte, error) {
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

// PublicKey returns the compressed 33-byte secp256k1 public key.
func (s *memSigner) PublicKey(account domain.Account) ([]byte, error) {
	priv, err := derivePrivKey(s.seed, account.Path)
	if err != nil {
		return nil, err
	}
	btcKey, _ := btcec.PrivKeyFromBytes(priv.D.FillBytes(make([]byte, 32)))
	return btcKey.PubKey().SerializeCompressed(), nil
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
