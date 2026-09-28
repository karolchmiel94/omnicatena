package keystore

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/karolchmiel94/omnicatena/internal/port"
)

// Postgres implements port.KeyStore over the keystore table (schema.sql) —
// same Argon2id+AES-256-GCM sealing as InMemory, persisted instead of held in
// a map. port.KeyStore takes no context, so these calls use context.Background().
type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (ks *Postgres) Create(walletID string, passphrase []byte) ([]byte, error) {
	seed, err := newSeed()
	if err != nil {
		return nil, err
	}

	enc, err := seal(seed, passphrase)
	if err != nil {
		return nil, err
	}

	_, err = ks.pool.Exec(context.Background(), `
		INSERT INTO keystore (wallet_id, salt, nonce, ciphertext) VALUES ($1, $2, $3, $4)
		ON CONFLICT (wallet_id) DO UPDATE
		SET salt = EXCLUDED.salt, nonce = EXCLUDED.nonce, ciphertext = EXCLUDED.ciphertext`,
		walletID, enc.salt, enc.nonce, enc.ciphertext)
	if err != nil {
		return nil, fmt.Errorf("keystore: save %s: %w", walletID, err)
	}
	return seed, nil
}

func (ks *Postgres) Unlock(walletID string, passphrase []byte) ([]byte, error) {
	var enc envelope
	err := ks.pool.QueryRow(context.Background(),
		`SELECT salt, nonce, ciphertext FROM keystore WHERE wallet_id = $1`, walletID,
	).Scan(&enc.salt, &enc.nonce, &enc.ciphertext)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("keystore: wallet %s not found", walletID)
		}
		return nil, fmt.Errorf("keystore: query %s: %w", walletID, err)
	}
	return open(enc, passphrase)
}

func (ks *Postgres) Signer(walletID string, passphrase []byte) (port.Signer, error) {
	seed, err := ks.Unlock(walletID, passphrase)
	if err != nil {
		return nil, err
	}
	return &memSigner{seed: seed}, nil
}
