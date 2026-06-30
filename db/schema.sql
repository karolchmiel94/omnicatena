-- Omnicatena V1 schema
-- Designed Day 5 (2026-06-30); implemented Day 6.
--
-- Design notes:
--   - chain is TEXT not enum: enums require migrations to add a value.
--   - amounts are NUMERIC(78, 0): holds any uint256 exactly, supports SQL
--     arithmetic, maps to big.Int via string. Smallest units throughout
--     (wei, sat, lamport, sun).
--   - keystore stores only encrypted blobs; plaintext seeds never persist.
--   - chain_cursors drives block-scanning monitoring: watchers resume from
--     block_height on restart instead of re-scanning from genesis.
--   - tx_addresses join table: decouples tx storage from wallet queries and
--     handles the edge case where both sides of a transfer are ours.

-- wallets ──────────────────────────────────────────────────────────────────

CREATE TABLE wallets (
    id          TEXT        PRIMARY KEY,   -- "w<unix_nano>"
    label       TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL
);

-- accounts ─────────────────────────────────────────────────────────────────
-- One row per (wallet, chain). Address format is chain-native:
--   EVM/Base  →  0x… (42 chars, checksum-cased)
--   Bitcoin   →  m… / n… (regtest P2PKH, base58check)
--   Solana    →  base58 pubkey (44 chars)
--   TRON      →  T… (base58check with 0x41 prefix, 34 chars)

CREATE TABLE accounts (
    wallet_id       TEXT NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    chain           TEXT NOT NULL,
    address         TEXT NOT NULL,
    derivation_path TEXT NOT NULL,
    PRIMARY KEY (wallet_id, chain)
);

-- Hot-path lookup during block scanning: "is this address one of ours?"
CREATE UNIQUE INDEX idx_accounts_chain_address ON accounts(chain, address);

-- keystore ─────────────────────────────────────────────────────────────────
-- Argon2id-stretched key + AES-256-GCM ciphertext. Plaintext seeds are
-- transient — they exist only in memory during derivation and signing.

CREATE TABLE keystore (
    wallet_id  TEXT  PRIMARY KEY REFERENCES wallets(id) ON DELETE CASCADE,
    salt       BYTEA NOT NULL,   -- 16 bytes, random per wallet
    nonce      BYTEA NOT NULL,   -- GCM nonce, random per seal
    ciphertext BYTEA NOT NULL    -- AES-256-GCM(seed, key=Argon2id(passphrase, salt))
);

-- chain_cursors ────────────────────────────────────────────────────────────
-- One row per chain. The watcher advances block_height after scanning each
-- block and uses it to resume without re-scanning after a restart.
-- block_hash is not used in V1 but anchors future reorg detection without
-- a schema change.

CREATE TABLE chain_cursors (
    chain        TEXT        PRIMARY KEY,
    block_height BIGINT      NOT NULL DEFAULT 0,
    block_hash   TEXT,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- transactions ─────────────────────────────────────────────────────────────
-- Every on-chain transaction that involves at least one of our addresses.
-- Inserted as 'pending' when first observed; updated to 'confirmed'/'failed'
-- once the block receipt is available.

CREATE TABLE transactions (
    id           BIGSERIAL      PRIMARY KEY,
    chain        TEXT           NOT NULL,
    hash         TEXT           NOT NULL,
    status       TEXT           NOT NULL
                     CHECK (status IN ('pending', 'confirmed', 'failed', 'unknown')),
    from_address TEXT,                        -- NULL for coinbase/mint txs
    to_address   TEXT,
    amount       NUMERIC(78, 0) NOT NULL,
    asset_symbol TEXT           NOT NULL,
    fee          NUMERIC(78, 0),              -- NULL until confirmed (receipt needed)
    block_height BIGINT,                      -- NULL while pending
    block_hash   TEXT,
    observed_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ,
    UNIQUE (chain, hash)
);

-- Only pending rows are live; confirmed/failed are terminal and excluded.
CREATE INDEX idx_transactions_pending
    ON transactions(chain, block_height)
    WHERE status = 'pending';

-- tx_addresses ─────────────────────────────────────────────────────────────
-- Links each transaction to the wallet account(s) involved.
-- Enables "all txs for wallet X on chain Y" as a single indexed lookup.

CREATE TABLE tx_addresses (
    tx_id     BIGINT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    wallet_id TEXT   NOT NULL REFERENCES wallets(id),
    chain     TEXT   NOT NULL,
    address   TEXT   NOT NULL,
    role      TEXT   NOT NULL CHECK (role IN ('sender', 'receiver')),
    PRIMARY KEY (tx_id, address, role)
);

CREATE INDEX idx_tx_addresses_wallet ON tx_addresses(wallet_id, chain);
