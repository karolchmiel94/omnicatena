// Package hdwallet provides BIP-32/44 and SLIP-0010 key derivation shared across chain adapters.
package hdwallet

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

// DeriveKey returns the extended key at path (e.g. "m/44'/60'/0'/0/0") derived from seed.
// chaincfg.MainNetParams is used as the root — its only effect here is on serialized key
// encoding (xpub/xprv prefixes); the BIP-32 math itself is chain-agnostic.
func DeriveKey(seed []byte, path domain.DerivationPath) (*hdkeychain.ExtendedKey, error) {
	master, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, fmt.Errorf("hdwallet: master key: %w", err)
	}
	indices, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	key := master
	for _, idx := range indices {
		key, err = key.Derive(idx)
		if err != nil {
			return nil, fmt.Errorf("hdwallet: derive index %d: %w", idx, err)
		}
	}
	return key, nil
}

// DeriveKeyEd25519 implements SLIP-0010 for ed25519. All path components must be
// hardened (index >= 2^31) — the spec forbids unhardened ed25519 derivation.
// Returns the 32-byte private key seed; pass to ed25519.NewKeyFromSeed.
func DeriveKeyEd25519(seed []byte, path domain.DerivationPath) ([]byte, error) {
	mac := hmac.New(sha512.New, []byte("ed25519 seed"))
	mac.Write(seed)
	I := mac.Sum(nil)
	key, chainCode := I[:32], I[32:]

	indices, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	for _, idx := range indices {
		if idx < hdkeychain.HardenedKeyStart {
			return nil, fmt.Errorf("hdwallet: ed25519 path requires all-hardened indices, got %d", idx)
		}
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], idx)

		data := make([]byte, 0, 37)
		data = append(data, 0x00)
		data = append(data, key...)
		data = append(data, b[:]...)

		h := hmac.New(sha512.New, chainCode)
		h.Write(data)
		I = h.Sum(nil)
		key, chainCode = I[:32], I[32:]
	}
	out := make([]byte, 32)
	copy(out, key)
	return out, nil
}

func parsePath(path domain.DerivationPath) ([]uint32, error) {
	s := string(path)
	if !strings.HasPrefix(s, "m/") {
		return nil, fmt.Errorf("hdwallet: path must start with m/: %q", s)
	}
	parts := strings.Split(s[2:], "/")
	out := make([]uint32, 0, len(parts))
	for _, p := range parts {
		hardened := strings.HasSuffix(p, "'")
		if hardened {
			p = p[:len(p)-1]
		}
		n, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("hdwallet: invalid path component %q", p)
		}
		idx := uint32(n)
		if hardened {
			idx += hdkeychain.HardenedKeyStart
		}
		out = append(out, idx)
	}
	return out, nil
}
