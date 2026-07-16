package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/yurifa/expense-tracker-api/internal/storage"

	"github.com/google/uuid"
)

func (s *Storage) CreateIdempotencyKey(ctx context.Context,
	params storage.CreateIdempotencyKeyParams,
) (*storage.IdempotencyKey, error) {
	const op = "storage.sqlite.CreateIdempotencyKey"

	stmt, err := s.db.PrepareContext(
		ctx,
		`INSERT INTO idempotency_keys (id, idempotency_key, user_id, request_hash, status, expires_at) VALUES (?, ?, ?, ?, ?, ?) RETURNING id, idempotency_key, user_id, request_hash, status, response_status, response_headers, response_body, created_at, updated_at, expires_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var idempotencyKey storage.IdempotencyKey
	id := uuid.NewString()
	err = stmt.QueryRowContext(ctx, id, params.IdempotencyKey, params.UserID, params.RequestHash, "pending", params.ExpiresAt.UTC()).
		Scan(
			&idempotencyKey.ID,
			&idempotencyKey.IdempotencyKey,
			&idempotencyKey.UserID,
			&idempotencyKey.RequestHash,
			&idempotencyKey.Status,
			&idempotencyKey.ResponseStatus,
			&idempotencyKey.ResponseHeaders,
			&idempotencyKey.ResponseBody,
			&idempotencyKey.CreatedAt,
			&idempotencyKey.UpdatedAt,
			&idempotencyKey.ExpiresAt,
		)
	if err != nil {
		if isUniqueConstraintViolation(err) {
			return nil, storage.ErrIdempotencyKeyInUse
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &idempotencyKey, nil
}

func (s *Storage) UpdateIdempotencyKey(ctx context.Context,
	userID string,
	id string,
	params storage.UpdateIdempotencyKeyParams,
) (*storage.IdempotencyKey, error) {
	const op = "storage.sqlite.UpdateIdempotencyKey"

	setParts, args := newUpdateBuilder().
		addString("status", params.Status).
		addInt("response_status", params.ResponseStatus).
		addBytes("response_headers", params.ResponseHeaders).
		addBytes("response_body", params.ResponseBody).
		build(", ")

	args = append(args, id)
	args = append(args, userID)

	query := fmt.Sprintf( //nolint:gosec // G201: setParts is built from a fixed whitelist, not user input
		`UPDATE idempotency_keys SET %s WHERE id = ? AND user_id = ? RETURNING id, idempotency_key, user_id, request_hash, status, response_status, response_headers, response_body, created_at, updated_at, expires_at`,
		setParts,
	)

	var idempotencyKey storage.IdempotencyKey
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&idempotencyKey.ID,
		&idempotencyKey.IdempotencyKey,
		&idempotencyKey.UserID,
		&idempotencyKey.RequestHash,
		&idempotencyKey.Status,
		&idempotencyKey.ResponseStatus,
		&idempotencyKey.ResponseHeaders,
		&idempotencyKey.ResponseBody,
		&idempotencyKey.CreatedAt,
		&idempotencyKey.UpdatedAt,
		&idempotencyKey.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrIdempotencyKeyNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &idempotencyKey, nil
}

func (s *Storage) GetByUserAndKey(ctx context.Context,
	userID string,
	key string,
) (*storage.IdempotencyKey, error) {
	const op = "storage.sqlite.GetByUserAndKey"

	query := `SELECT id, idempotency_key, user_id, request_hash, status, response_status, response_headers, response_body, created_at, updated_at, expires_at FROM idempotency_keys WHERE user_id = ? AND idempotency_key = ?`

	var idempotencyKey storage.IdempotencyKey
	err := s.db.QueryRowContext(ctx, query, userID, key).Scan(
		&idempotencyKey.ID,
		&idempotencyKey.IdempotencyKey,
		&idempotencyKey.UserID,
		&idempotencyKey.RequestHash,
		&idempotencyKey.Status,
		&idempotencyKey.ResponseStatus,
		&idempotencyKey.ResponseHeaders,
		&idempotencyKey.ResponseBody,
		&idempotencyKey.CreatedAt,
		&idempotencyKey.UpdatedAt,
		&idempotencyKey.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrIdempotencyKeyNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &idempotencyKey, nil
}

func (s *Storage) DeleteIdempotencyKey(ctx context.Context, userID string, id string) error {
	const op = "storage.sqlite.DeleteIdempotencyKey"

	query := `DELETE FROM idempotency_keys WHERE id = ? AND user_id = ?`

	_, err := s.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) DeleteExpiredIdempotencyKeys(ctx context.Context) (int64, error) {
	const op = "storage.sqlite.DeleteExpiredIdempotencyKeys"

	query := `DELETE FROM idempotency_keys WHERE expires_at <= CURRENT_TIMESTAMP`

	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return rowsAffected, nil
}
