-- +goose Up
-- SQLite can't ALTER a CHECK constraint, so we recreate the table.
-- Old values: sensor → ESP32, api_integration → ExternalAPI, manual → ExternalAPI
CREATE TABLE sources_new (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    kind       TEXT NOT NULL CHECK (kind IN ('ESP32', 'ExternalAPI')),
    config     TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

INSERT INTO sources_new (id, name, kind, config, created_at)
SELECT id, name,
    CASE kind
        WHEN 'sensor'          THEN 'ESP32'
        WHEN 'api_integration' THEN 'ExternalAPI'
        ELSE                        'ExternalAPI'
    END,
    config, created_at
FROM sources;

DROP TABLE sources;
ALTER TABLE sources_new RENAME TO sources;

-- +goose Down
CREATE TABLE sources_old (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    kind       TEXT NOT NULL CHECK (kind IN ('sensor', 'api_integration', 'manual')),
    config     TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

INSERT INTO sources_old (id, name, kind, config, created_at)
SELECT id, name,
    CASE kind
        WHEN 'ESP32'       THEN 'sensor'
        WHEN 'ExternalAPI' THEN 'api_integration'
        ELSE                    'sensor'
    END,
    config, created_at
FROM sources;

DROP TABLE sources;
ALTER TABLE sources_old RENAME TO sources;
