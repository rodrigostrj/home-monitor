-- +goose Up

CREATE TABLE sources (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    kind       TEXT NOT NULL CHECK (kind IN ('sensor', 'api_integration', 'manual')),
    api_key    TEXT NOT NULL,
    config     TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE readings (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id   TEXT    NOT NULL REFERENCES sources (id),
    metric_type TEXT    NOT NULL,
    value       REAL    NOT NULL,
    unit        TEXT    NOT NULL,
    recorded_at DATETIME NOT NULL,
    received_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    metadata    TEXT
);

CREATE INDEX readings_source_metric_time
    ON readings (source_id, metric_type, recorded_at DESC);

-- +goose Down

DROP TABLE readings;
DROP TABLE sources;
