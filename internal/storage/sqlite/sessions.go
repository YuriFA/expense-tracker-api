package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"expense-tracker-api/internal/storage"
)

func (s *Storage) CreateSession(params storage.CreateSessionParams) (*storage.Session, error) {
	const op = "storage.sqlite.CreateSession"

	stmt, err := s.db.Prepare(
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?) RETURNING id, user_id, expires_at, created_at, updated_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var session storage.Session
	err = stmt.QueryRow(params.SessionID, params.UserID, params.ExpiresAt).
		Scan(&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &session, nil
}

func (s *Storage) GetSessionByID(id string) (*storage.Session, error) {
	const op = "storage.sqlite.GetSessionByID"

	stmt, err := s.db.Prepare(
		`SELECT id, user_id, expires_at, created_at, updated_at FROM sessions WHERE id = ? AND expires_at > CURRENT_TIMESTAMP`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var session storage.Session
	err = stmt.QueryRow(id).
		Scan(&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrSessionNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &session, nil
}

func (s *Storage) DeleteSession(id string) error {
	const op = "storage.sqlite.DeleteSession"

	stmt, err := s.db.Prepare(
		`DELETE FROM sessions WHERE id = ?`,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(id)
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

func (s *Storage) ExtendSession(id string, newExpiresAt time.Time) error {
	const op = "storage.sqlite.ExtendSession"

	stmt, err := s.db.Prepare(
		`UPDATE sessions SET expires_at = ? WHERE id = ? AND expires_at > CURRENT_TIMESTAMP`,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(newExpiresAt, id)
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

func (s *Storage) DeleteExpiredSessions() (int64, error) {
	const op = "storage.sqlite.DeleteExpiredSessions"

	res, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP`)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return rowsAffected, nil
}
