-- reverse 000002_add_users.up.sql

ALTER TABLE IF EXISTS accounts DROP CONSTRAINT IF EXISTS accounts_owner_fkey;

ALTER TABLE IF EXISTS accounts DROP CONSTRAINT IF EXISTS accounts_owner_currency_key;

DROP TABLE IF EXISTS users CASCADE;
