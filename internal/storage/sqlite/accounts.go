package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"expense-tracker-api/internal/storage"

	"github.com/google/uuid"
)

func (s *Storage) CreateAccount(name string, openingBalance float64) (*storage.Account, error) {
	const op = "storage.sqlite.CreateAccount"

	stmt, err := s.db.Prepare(
		`INSERT INTO accounts (id, name, opening_balance, manual_adjustment) VALUES (?, ?, ?, ?) RETURNING id, name, opening_balance, manual_adjustment, created_at, updated_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	id := uuid.NewString()
	var account storage.Account
	err = stmt.QueryRow(id, name, openingBalance, 0.0).
		Scan(&account.Id, &account.Name, &account.OpeningBalance, &account.ManualAdjustment, &account.CreatedAt, &account.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &account, nil
}

func (s *Storage) UpdateAccount(
	id string,
	params storage.UpdateAccountParams,
) (*storage.Account, error) {
	const op = "storage.sqlite.UpdateAccount"

	setParts, args := newUpdateBuilder().
		addString("name", params.Name).
		addFloat("manual_adjustment", params.ManualAdjustment).
		build(", ")

	args = append(args, id)

	query := fmt.Sprintf(
		`UPDATE accounts SET %s WHERE id = ? RETURNING id, name, opening_balance, manual_adjustment, created_at, updated_at`,
		setParts,
	)

	var account storage.Account
	err := s.db.QueryRow(query, args...).
		Scan(&account.Id, &account.Name, &account.OpeningBalance, &account.ManualAdjustment, &account.CreatedAt, &account.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrAccountNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &account, nil
}

func (s *Storage) DeleteAccount(id string) error {
	const op = "storage.sqlite.DeleteAccount"

	stmt, err := s.db.Prepare(
		`DELETE FROM accounts WHERE id = ?`,
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
		return fmt.Errorf("%s: %w", op, storage.ErrAccountNotFound)
	}

	return nil
}

func (s *Storage) GetAccount(id string) (*storage.Account, error) {
	const op = "storage.sqlite.GetAccount"

	stmt, err := s.db.Prepare(
		`SELECT id, name, opening_balance, manual_adjustment, created_at, updated_at FROM accounts WHERE id = ?`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var account storage.Account
	err = stmt.QueryRow(id).
		Scan(&account.Id, &account.Name, &account.OpeningBalance, &account.ManualAdjustment, &account.CreatedAt, &account.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrAccountNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &account, nil
}

func (s *Storage) GetAccounts() ([]storage.Account, error) {
	const op = "storage.sqlite.GetAccounts"

	stmt, err := s.db.Prepare(
		`SELECT id, name, opening_balance, manual_adjustment, created_at, updated_at FROM accounts`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	accounts := []storage.Account{}
	rows, err := stmt.Query()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		account := storage.Account{}
		err := rows.Scan(
			&account.Id,
			&account.Name,
			&account.OpeningBalance,
			&account.ManualAdjustment,
			&account.CreatedAt,
			&account.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return accounts, nil
}
