FROM golang:1.26-alpine AS builder

RUN apk add --no-cache \
  # Important: required for go-sqlite3
  gcc \
  # Required for Alpine
  musl-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/expense-tracker-api ./cmd/expense-tracker-api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/expense-tracker-api /app/expense-tracker-api
COPY config/prod.yaml /app/prod.yaml
# for CLI migrate command
COPY internal/storage/sqlite/migrations /app/migrations
RUN mkdir -p /app/storage
EXPOSE 8080
ENV ENV=prod
ENV TZ=UTC
ENV CONFIG_PATH=/app/prod.yaml
ENV STORAGE_PATH=/app/storage/storage.db
ENTRYPOINT [ "/app/expense-tracker-api" ]
