package domain

import "time"

// Address format and validation are owned by the chain adapter, not the domain.
type Address string

type DerivationPath string

type Account struct {
	Chain   ChainID
	Address Address
	Path    DerivationPath
}

// The wallet's seed/mnemonic is never stored here — it is held encrypted by the
// KeyStore and referenced by ID (ADR-0004).
type Wallet struct {
	ID        string
	Label     string
	CreatedAt time.Time
	Accounts  []Account
}

func (w Wallet) Account(chain ChainID) (Account, bool) {
	for _, a := range w.Accounts {
		if a.Chain == chain {
			return a, true
		}
	}
	return Account{}, false
}
