-- name: InsertSource :one
INSERT INTO sources (id, name, kind, api_key, config)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetSourceByID :one
SELECT * FROM sources WHERE id = ?;

-- name: GetSourceByAPIKey :one
SELECT * FROM sources WHERE api_key = ?;

-- name: ListSources :many
SELECT * FROM sources ORDER BY created_at ASC;
