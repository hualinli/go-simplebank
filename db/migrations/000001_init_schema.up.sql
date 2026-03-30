-- init schema for simple-bank

CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    owner TEXT NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0,
    currency TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounts_owner ON accounts(owner);
CREATE INDEX idx_accounts_currency ON accounts(currency);

CREATE TABLE entries (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_entries_account_id ON entries(account_id);

CREATE TABLE transfers (
    id BIGSERIAL PRIMARY KEY,
    from_account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    to_account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    amount BIGINT NOT NULL CHECK (amount > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (from_account_id <> to_account_id)
);

CREATE INDEX idx_transfers_from_account_id ON transfers(from_account_id);
CREATE INDEX idx_transfers_to_account_id ON transfers(to_account_id);

