package keystore

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/karolchmiel94/omnicatena/internal/domain"
	"github.com/karolchmiel94/omnicatena/internal/hdwallet"
)

type memSigner struct {
	seed []byte
}

func (s *memSigner) Sign(account domain.Account, digest []byte) ([]byte, error) {
	priv, err := derivePrivKey(s.seed, account.Path)
	if err != nil {
		return nil, err
	}
	return crypto.Sign(digest, priv)
}

func (s *memSigner) PublicKey(account domain.Account) ([]byte, error) {
	priv, err := derivePrivKey(s.seed, account.Path)
	if err != nil {
		return nil, err
	}
	return crypto.FromECDSAPub(&priv.PublicKey), nil
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
