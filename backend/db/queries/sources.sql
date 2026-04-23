-- name: InsertSource :one
INSERT INTO sources (id, name, kind, config)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetSourceByID :one
SELECT * FROM sources WHERE id = ?;

-- name: ListSources :many
SELECT * FROM sources ORDER BY created_at ASC;
