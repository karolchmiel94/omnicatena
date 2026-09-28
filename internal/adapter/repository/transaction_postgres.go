package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

// PostgresTransaction implements port.TransactionRepository over the
// transactions + tx_addresses tables (schema.sql). Chain-agnostic: chain is a
// column value, not a table, so this one implementation serves every chain a
// ChainWatcher is added for.
type PostgresTransaction struct {
	pool *pgxpool.Pool
}

func NewPostgresTransaction(pool *pgxpool.Pool) *PostgresTransaction {
	return &PostgresTransaction{pool: pool}
}

func (r *PostgresTransaction) SaveEvent(ctx context.Context, walletID string, evt domain.TxEvent) error {
	from, to := fromTo(evt)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repository: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var txID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (chain, hash, status, from_address, to_address, amount, asset_symbol, block_height, confirmed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CASE WHEN $3 = 'confirmed' THEN NOW() ELSE NULL END)
		ON CONFLICT (chain, hash) DO UPDATE
		SET status = EXCLUDED.status, block_height = EXCLUDED.block_height, confirmed_at = EXCLUDED.confirmed_at
		RETURNING id`,
		string(evt.Chain), evt.Hash, string(evt.Status), from, to,
		evt.Amount.Base.String(), evt.Amount.Asset.Symbol, blockHeightOrNil(evt.BlockHeight),
	).Scan(&txID)
	if err != nil {
		return fmt.Errorf("repository: save transaction %s/%s: %w", evt.Chain, evt.Hash, err)
	}

	if role, ok := roleFor(evt.Type); ok {
		_, err = tx.Exec(ctx, `
			INSERT INTO tx_addresses (tx_id, wallet_id, chain, address, role)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (tx_id, address, role) DO NOTHING`,
			txID, walletID, string(evt.Chain), string(evt.Address), role)
		if err != nil {
			return fmt.Errorf("repository: link tx_addresses %s/%s: %w", evt.Chain, evt.Hash, err)
		}
	}

	return tx.Commit(ctx)
}

// fromTo maps a TxEvent's (watched-address, direction) pair onto the
// transactions table's plain from_address/to_address columns.
func fromTo(evt domain.TxEvent) (from, to *string) {
	addr := string(evt.Address)
	switch evt.Type {
	case domain.EventOutbound:
		return &addr, nonEmptyPtr(string(evt.Counterparty))
	case domain.EventInbound:
		return nonEmptyPtr(string(evt.Counterparty)), &addr
	default:
		return nil, nil
	}
}

func nonEmptyPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func blockHeightOrNil(h uint64) *int64 {
	if h == 0 {
		return nil
	}
	v := int64(h)
	return &v
}

// roleFor maps a TxEvent's direction to the tx_addresses.role CHECK constraint
// ('sender'/'receiver'). ok is false for event types that aren't a wallet's
// own side of a transfer (e.g. domain.EventConfirmed — not yet emitted by any
// watcher, but handled here rather than assumed away).
func roleFor(t domain.TxEventType) (role string, ok bool) {
	switch t {
	case domain.EventOutbound:
		return "sender", true
	case domain.EventInbound:
		return "receiver", true
	default:
		return "", false
	}
}
