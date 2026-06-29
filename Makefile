-include .env
export CONFIG_PATH

.PHONY: run build tidy test

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
