.PHONY: run build migrate sqlc test

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

migrate:
	goose -dir db/migrations sqlite3 home-monitor.db up

sqlc:
	sqlc generate

test:
	go test ./...
