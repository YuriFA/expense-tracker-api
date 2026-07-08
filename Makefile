-include .env
export CONFIG_PATH
export DATABASE_URL

.PHONY: run build tidy test migrate-up migrate-down migrate-create dev

run:
	@echo "Starting Expense Tracker API..."
	go run ./cmd/expense-tracker-api

build:
	@echo "Building binary..."
	go build -o bin/expense-tracker-api ./cmd/expense-tracker-api

tidy:
	go mod tidy

dev: tidy run

test:
	go test ./...

migrate-up:
	migrate -path ./internal/storage/sqlite/migrations/ -database "sqlite3://$(DATABASE_URL)" up

migrate-down:
	migrate -path ./internal/storage/sqlite/migrations/ -database "sqlite3://$(DATABASE_URL)" down

migrate-down-all:
	migrate -path ./internal/storage/sqlite/migrations/ -database "sqlite3://$(DATABASE_URL)" down --all

migrate-create:
	@test -n "$(name)" || { echo "Usage: make migrate-create name=add_users"; exit 1; }
	@migrate create -ext sql -dir ./internal/storage/sqlite/migrations/ -seq $(name)

