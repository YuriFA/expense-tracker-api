package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/yurifa/expense-tracker-api/internal/storage"

	"github.com/google/uuid"
	"github.com/mattn/go-sqlite3"
)

func (s *Storage) RegisterUser(ctx context.Context, params storage.RegisterUserParams) (*storage.User, error) {
	const op = "storage.sqlite.RegisterUser"

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = tx.Rollback() }()

	var user storage.User
	err = tx.QueryRowContext(ctx, `INSERT INTO users (id, email, password_hash)
		VALUES (?, ?, ?)
		RETURNING id, email, created_at, updated_at`,
		uuid.NewString(), params.Email, params.PasswordHash,
	).Scan(&user.ID, &user.Email, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrUserAlreadyExists)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for _, category := range defaultCategories {
		_, err = tx.ExecContext(ctx, `INSERT INTO categories (id, user_id, name, type, icon, color)
               VALUES (?, ?, ?, ?, ?, ?)`,
			uuid.NewString(), user.ID, category.Name, category.Type, category.Icon, category.Color,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}

func (s *Storage) GetUserByEmail(ctx context.Context, email string) (*storage.User, error) {
	const op = "storage.sqlite.GetUserByEmail"

	stmt, err := s.db.PrepareContext(
		ctx,
		`SELECT id, email, password_hash, created_at, updated_at FROM users WHERE email = ?`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var user storage.User
	err = stmt.QueryRowContext(ctx, email).
		Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}

func (s *Storage) GetUserByID(ctx context.Context, id string) (*storage.User, error) {
	const op = "storage.sqlite.GetUserByID"

	stmt, err := s.db.PrepareContext(ctx, `SELECT id, email, created_at, updated_at FROM users WHERE id = ?`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var user storage.User
	err = stmt.QueryRowContext(ctx, id).Scan(&user.ID, &user.Email, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}
