-- name: InsertReading :exec
INSERT INTO readings (source_id, metric_type, value, unit, recorded_at, metadata)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetLatestReadingsByMetric :many
-- Returns the most recent reading per source for a given metric type.
WITH latest AS (
    SELECT source_id, MAX(recorded_at) AS max_at
    FROM readings
    WHERE metric_type = sqlc.arg(metric_type)
    GROUP BY source_id
)
SELECT r.id, r.source_id, r.metric_type, r.value, r.unit, r.recorded_at, r.received_at, r.metadata
FROM readings r
JOIN latest ON r.source_id = latest.source_id AND r.recorded_at = latest.max_at
WHERE r.metric_type = sqlc.arg(metric_type);

-- name: GetReadingsInRange :many
SELECT * FROM readings
WHERE source_id   = sqlc.arg(source_id)
  AND metric_type = sqlc.arg(metric_type)
  AND recorded_at >= sqlc.arg(from_time)
  AND recorded_at <= sqlc.arg(to_time)
ORDER BY recorded_at ASC;
