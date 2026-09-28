package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/karolchmiel94/omnicatena/internal/domain"
)

// PostgresWallet implements port.WalletRepository over the wallets + accounts
// tables (schema.sql). Chain-agnostic: chain is a column value on accounts,
// not a table — one implementation serves every chain.
type PostgresWallet struct {
	pool *pgxpool.Pool
}

func NewPostgresWallet(pool *pgxpool.Pool) *PostgresWallet {
	return &PostgresWallet{pool: pool}
}

func (r *PostgresWallet) Save(ctx context.Context, w domain.Wallet) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repository: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO wallets (id, label, created_at) VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET label = EXCLUDED.label`,
		w.ID, w.Label, w.CreatedAt)
	if err != nil {
		return fmt.Errorf("repository: save wallet %s: %w", w.ID, err)
	}

	for _, acc := range w.Accounts {
		_, err = tx.Exec(ctx, `
			INSERT INTO accounts (wallet_id, chain, address, derivation_path)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (wallet_id, chain) DO UPDATE
			SET address = EXCLUDED.address, derivation_path = EXCLUDED.derivation_path`,
			w.ID, string(acc.Chain), string(acc.Address), string(acc.Path))
		if err != nil {
			return fmt.Errorf("repository: save account %s/%s: %w", w.ID, acc.Chain, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresWallet) Get(ctx context.Context, id string) (domain.Wallet, error) {
	var w domain.Wallet
	err := r.pool.QueryRow(ctx, `SELECT id, label, created_at FROM wallets WHERE id = $1`, id).
		Scan(&w.ID, &w.Label, &w.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Wallet{}, fmt.Errorf("repository: wallet %s not found", id)
		}
		return domain.Wallet{}, fmt.Errorf("repository: get wallet %s: %w", id, err)
	}

	accounts, err := r.accountsFor(ctx, id)
	if err != nil {
		return domain.Wallet{}, err
	}
	w.Accounts = accounts
	return w, nil
}

func (r *PostgresWallet) List(ctx context.Context) ([]domain.Wallet, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, label, created_at FROM wallets`)
	if err != nil {
		return nil, fmt.Errorf("repository: list wallets: %w", err)
	}
	defer rows.Close()

	var wallets []domain.Wallet
	for rows.Next() {
		var w domain.Wallet
		if err := rows.Scan(&w.ID, &w.Label, &w.CreatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan wallet: %w", err)
		}
		wallets = append(wallets, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: list wallets: %w", err)
	}

	// N+1 on purpose — V1 wallet counts are small; revisit with a JOIN if that changes.
	for i := range wallets {
		accounts, err := r.accountsFor(ctx, wallets[i].ID)
		if err != nil {
			return nil, err
		}
		wallets[i].Accounts = accounts
	}
	return wallets, nil
}

func (r *PostgresWallet) accountsFor(ctx context.Context, walletID string) ([]domain.Account, error) {
	rows, err := r.pool.Query(ctx, `SELECT chain, address, derivation_path FROM accounts WHERE wallet_id = $1`, walletID)
	if err != nil {
		return nil, fmt.Errorf("repository: list accounts for %s: %w", walletID, err)
	}
	defer rows.Close()

	var accounts []domain.Account
	for rows.Next() {
		var chain, address, path string
		if err := rows.Scan(&chain, &address, &path); err != nil {
			return nil, fmt.Errorf("repository: scan account: %w", err)
		}
		accounts = append(accounts, domain.Account{
			Chain:   domain.ChainID(chain),
			Address: domain.Address(address),
			Path:    domain.DerivationPath(path),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: list accounts for %s: %w", walletID, err)
	}
	return accounts, nil
}
