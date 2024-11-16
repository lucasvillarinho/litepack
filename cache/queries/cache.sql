-- name: GetValue :one
SELECT value
FROM cache
WHERE key = ? AND expires_at > ?;

-- name: UpdateLastAccessedAt :exec
UPDATE cache
SET last_accessed_at = ?
WHERE key = ?;


-- name: DeleteKey :exec
DELETE FROM cache
WHERE key = ?;


-- name: CountEntries :one
SELECT COUNT(*)
FROM cache;

-- name: SelectKeysToDelete :many
SELECT key
FROM cache
ORDER BY last_accessed_at ASC
LIMIT ?;

-- name: DeleteKeys :exec
DELETE FROM cache
WHERE key IN (
    SELECT key
    FROM cache
    ORDER BY last_accessed_at ASC
    LIMIT ?
);

-- name: CreateDatabase :exec
CREATE TABLE IF NOT EXISTS cache (
    key TEXT PRIMARY KEY,
    value BLOB,
    expires_at TIMESTAMP NOT NULL,
    last_accessed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);


-- name: UpsertCache :exec
INSERT INTO cache (key, value, expires_at, last_accessed_at)
VALUES (?, ?, ?, ?)
ON CONFLICT (key) DO UPDATE
SET value = excluded.value,
    expires_at = excluded.expires_at,
    last_accessed_at = excluded.last_accessed_at;