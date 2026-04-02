-- add users table for simple-bank

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    hashed_password TEXT NOT NULL,
    full_name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_changed_at TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01T00:00:00Z',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE accounts ADD FOREIGN KEY (owner) REFERENCES users (username) ON DELETE CASCADE;

ALTER TABLE accounts ADD CONSTRAINT accounts_owner_currency_key UNIQUE (owner, currency);