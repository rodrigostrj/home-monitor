package storage

import (
	"database/sql"
	"fmt"

	homedb "github.com/rodrigo/home-monitor/db"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

// Open opens (or creates) the SQLite database at path and runs any pending
// goose migrations. The binary is self-contained: migrations are embedded.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	goose.SetBaseFS(homedb.Migrations)
	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect("sqlite3"); err != nil {
		return nil, fmt.Errorf("goose dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return nil, fmt.Errorf("goose up: %w", err)
	}

	return db, nil
}
