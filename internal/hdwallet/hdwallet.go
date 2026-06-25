// Package hdwallet provides BIP-32/44 key derivation shared across chain adapters.
package hdwallet

import (
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
