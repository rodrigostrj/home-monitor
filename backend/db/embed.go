package db

import "embed"

// Migrations holds all goose migration files embedded into the binary,
// so the deployed binary is self-contained — no migrations directory needed on the server.
//go:embed migrations/*.sql
var Migrations embed.FS
