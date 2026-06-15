package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"expense-tracker-api/internal/storage"

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
			id INTEGER PRIMARY KEY,
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
	Id               int64   `json:"id"`
	Name             string  `json:"name"`
	OpeningBalance   float64 `json:"openingBalance"`
	ManualAdjustment float64 `json:"manualAdjustment"`
}

func (s *Storage) GetAccounts() ([]Account, error) {
	const op = "storage.sqlite.GetAccounts"

	stmt, err := s.db.Prepare(`SELECT id, name, opening_balance, manual_adjustment FROM accounts`)
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

func (s *Storage) CreateAccount(name string, openingBalance float64) (*Account, error) {
	const op = "storage.sqlite.CreateAccount"

	stmt, err := s.db.Prepare(
		`INSERT INTO accounts (name, opening_balance, manual_adjustment) VALUES (?, ?, ?) RETURNING id, name, opening_balance, manual_adjustment`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var account Account
	err = stmt.QueryRow(name, openingBalance, 0.0).
		Scan(&account.Id, &account.Name, &account.OpeningBalance, &account.ManualAdjustment)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrAccountNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &account, nil
}

func (s *Storage) GetAccount(id int64) (*Account, error) {
	const op = "storage.sqlite.GetAccount"

	stmt, err := s.db.Prepare(
		`SELECT id, name, opening_balance, manual_adjustment FROM accounts WHERE id = ?`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var account Account
	err = stmt.QueryRow(id).
		Scan(&account.Id, &account.Name, &account.OpeningBalance, &account.ManualAdjustment)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrAccountNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &account, nil
}

func (s *Storage) DeleteAccount(id int64) error {
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
