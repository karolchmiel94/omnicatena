package keystore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/karolchmiel94/omnicatena/internal/port"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 1
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	saltLen      = 16
)

type envelope struct {
	salt       []byte
	nonce      []byte
	ciphertext []byte
}

// InMemory stores encrypted seeds in memory (no persistence in V1).
type InMemory struct {
	mu    sync.RWMutex
	store map[string]envelope
}

func New() *InMemory {
	return &InMemory{store: make(map[string]envelope)}
}

func (ks *InMemory) Create(walletID string, passphrase []byte) ([]byte, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return nil, err
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return nil, err
	}
	seed := bip39.NewSeed(mnemonic, "")

	enc, err := seal(seed, passphrase)
	if err != nil {
		return nil, err
	}

	ks.mu.Lock()
	ks.store[walletID] = enc
	ks.mu.Unlock()

	return seed, nil
}

func (ks *InMemory) Unlock(walletID string, passphrase []byte) ([]byte, error) {
	ks.mu.RLock()
	enc, ok := ks.store[walletID]
	ks.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("keystore: wallet %s not found", walletID)
	}
	return open(enc, passphrase)
}

func (ks *InMemory) Signer(walletID string, passphrase []byte) (port.Signer, error) {
	seed, err := ks.Unlock(walletID, passphrase)
	if err != nil {
		return nil, err
	}
	return &memSigner{seed: seed}, nil
}

func seal(plaintext, passphrase []byte) (envelope, error) {
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return envelope{}, err
	}
	key := stretchKey(passphrase, salt)
	gcm, err := newGCM(key)
	if err != nil {
		return envelope{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return envelope{}, err
	}
	return envelope{
		salt:       salt,
		nonce:      nonce,
		ciphertext: gcm.Seal(nil, nonce, plaintext, nil),
	}, nil
}

func open(enc envelope, passphrase []byte) ([]byte, error) {
	key := stretchKey(passphrase, enc.salt)
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	plaintext, err := gcm.Open(nil, enc.nonce, enc.ciphertext, nil)
	if err != nil {
		return nil, errors.New("keystore: invalid passphrase")
	}
	return plaintext, nil
}

func stretchKey(passphrase, salt []byte) []byte {
	return argon2.IDKey(passphrase, salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
