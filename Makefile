include .env
export

.PHONY: run build tidy

run:
	@echo "Starting Expense Tracker API..."
	go run ./cmd/expense-tracker-api

build:
	@echo "Building binary..."
	go build -o bin/expense-tracker-api ./cmd/expense-tracker-api

tidy:
	go mod tidy

dev: tidy run
