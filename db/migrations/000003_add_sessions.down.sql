-- reverse 000003_add_sessions.up.sql

DROP INDEX IF EXISTS idx_sessions_username;
DROP INDEX IF EXISTS idx_sessions_expires_at;

DROP TABLE IF EXISTS sessions CASCADE;
