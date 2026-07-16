package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/yurifa/expense-tracker-api/internal/storage"
)

func (s *Storage) CreateSession(ctx context.Context, params storage.CreateSessionParams) (*storage.Session, error) {
	const op = "storage.sqlite.CreateSession"

	stmt, err := s.db.PrepareContext(
		ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?) RETURNING id, user_id, expires_at, created_at, updated_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var session storage.Session
	err = stmt.QueryRowContext(ctx, params.SessionID, params.UserID, params.ExpiresAt).
		Scan(&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &session, nil
}

func (s *Storage) GetSessionByID(ctx context.Context, id string) (*storage.Session, error) {
	const op = "storage.sqlite.GetSessionByID"

	stmt, err := s.db.PrepareContext(
		ctx,
		`SELECT id, user_id, expires_at, created_at, updated_at FROM sessions WHERE id = ? AND expires_at > CURRENT_TIMESTAMP`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var session storage.Session
	err = stmt.QueryRowContext(ctx, id).
		Scan(&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrSessionNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &session, nil
}

func (s *Storage) DeleteSession(ctx context.Context, id string) error {
	const op = "storage.sqlite.DeleteSession"

	stmt, err := s.db.PrepareContext(ctx, `DELETE FROM sessions WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrSessionNotFound)
	}

	return nil
}

func (s *Storage) ExtendSession(ctx context.Context, id string, newExpiresAt time.Time) error {
	const op = "storage.sqlite.ExtendSession"

	stmt, err := s.db.PrepareContext(
		ctx,
		`UPDATE sessions SET expires_at = ? WHERE id = ? AND expires_at > CURRENT_TIMESTAMP`,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, newExpiresAt, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrSessionNotFound)
	}

	return nil
}

func (s *Storage) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	const op = "storage.sqlite.DeleteExpiredSessions"

	res, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP`)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return rowsAffected, nil
}
