package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

// PostgresCursor implements port.ChainCursorStore over chain_cursors
// (schema.sql) — one row per chain, so it serves every chain a ChainWatcher
// is added for.
type PostgresCursor struct {
	pool *pgxpool.Pool
}

func NewPostgresCursor(pool *pgxpool.Pool) *PostgresCursor {
	return &PostgresCursor{pool: pool}
}

func (r *PostgresCursor) Get(ctx context.Context, chain domain.ChainID) (uint64, bool, error) {
	var height int64
	err := r.pool.QueryRow(ctx, `SELECT block_height FROM chain_cursors WHERE chain = $1`, string(chain)).Scan(&height)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("repository: get cursor %s: %w", chain, err)
	}
	return uint64(height), true, nil
}

func (r *PostgresCursor) Set(ctx context.Context, chain domain.ChainID, height uint64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO chain_cursors (chain, block_height, updated_at) VALUES ($1, $2, NOW())
		ON CONFLICT (chain) DO UPDATE SET block_height = EXCLUDED.block_height, updated_at = NOW()`,
		string(chain), int64(height))
	if err != nil {
		return fmt.Errorf("repository: set cursor %s: %w", chain, err)
	}
	return nil
}
