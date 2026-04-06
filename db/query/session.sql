-- name: CreateSession :one
INSERT INTO sessions (
    session_id,
    username,
    refresh_token,
    user_agent,
    client_ip,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetSession :one
SELECT * FROM sessions WHERE session_id = $1 LIMIT 1;