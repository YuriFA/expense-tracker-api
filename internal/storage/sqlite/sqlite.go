package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"expense-tracker-api/internal/storage"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3" // init sqlite3 driver
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err := db.Prepare(`
		CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			opening_balance REAL NOT NULL,
			manual_adjustment REAL NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	defer stmt.Close()
	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

type Account struct {
	Id               string  `json:"id"`
	Name             string  `json:"name"`
	OpeningBalance   float64 `json:"openingBalance"`
	ManualAdjustment float64 `json:"manualAdjustment"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
}

type UpdateAccountParams struct {
	Name             *string
	ManualAdjustment *float64
}

func (s *Storage) CreateAccount(name string, openingBalance float64) (*Account, error) {
	const op = "storage.sqlite.CreateAccount"

	stmt, err := s.db.Prepare(
		`INSERT INTO accounts (id, name, opening_balance, manual_adjustment) VALUES (?, ?, ?, ?) RETURNING id, name, opening_balance, manual_adjustment, created_at, updated_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	id := uuid.NewString()
	var account Account
	err = stmt.QueryRow(id, name, openingBalance, 0.0).
		Scan(&account.Id, &account.Name, &account.OpeningBalance, &account.ManualAdjustment, &account.CreatedAt, &account.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrAccountNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &account, nil
}

func (s *Storage) UpdateAccount(
	id string,
	params UpdateAccountParams,
) (*Account, error) {
	const op = "storage.sqlite.UpdateAccount"

	setParts := []string{"updated_at = CURRENT_TIMESTAMP"}
	args := []any{}
	if params.Name != nil {
		setParts = append(setParts, "name = ?")
		args = append(args, *params.Name)
	}
	if params.ManualAdjustment != nil {
		setParts = append(setParts, "manual_adjustment = ?")
		args = append(args, *params.ManualAdjustment)
	}

	args = append(args, id)

	query := fmt.Sprintf(
		`UPDATE accounts SET %s WHERE id = ? RETURNING id, name, opening_balance, manual_adjustment, created_at, updated_at`,
		strings.Join(setParts, ", "),
	)

	var account Account
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

	_, err = stmt.Exec(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%s: %w", op, storage.ErrAccountNotFound)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) GetAccount(id string) (*Account, error) {
	const op = "storage.sqlite.GetAccount"

	stmt, err := s.db.Prepare(
		`SELECT id, name, opening_balance, manual_adjustment, created_at, updated_at FROM accounts WHERE id = ?`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var account Account
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

func (s *Storage) GetAccounts() ([]Account, error) {
	const op = "storage.sqlite.GetAccounts"

	stmt, err := s.db.Prepare(
		`SELECT id, name, opening_balance, manual_adjustment, created_at, updated_at FROM accounts`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var accounts []Account
	rows, err := stmt.Query()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		account := Account{}
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
