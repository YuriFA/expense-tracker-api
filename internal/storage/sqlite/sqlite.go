package sqlite

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3" // init sqlite3 driver
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	db.SetMaxOpenConns(1)

	_, err = db.Exec(`
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

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT UNIQUE,
			type TEXT NOT NULL CHECK(type IN ('income', 'expense')),
			icon TEXT NOT NULL,
			color TEXT NOT NULL,
			is_default INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS transactions (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL CHECK(type IN ('income', 'expense', 'transfer')),
			amount REAL NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			occurred_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			account_id TEXT,
			category_id TEXT,
			from_account_id TEXT,
			to_account_id TEXT,
			FOREIGN KEY (account_id) REFERENCES accounts(id),
			FOREIGN KEY (category_id) REFERENCES categories(id),
			FOREIGN KEY (from_account_id) REFERENCES accounts(id),
			FOREIGN KEY (to_account_id) REFERENCES accounts(id)
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = db.Exec(`
		CREATE VIEW IF NOT EXISTS account_contributions AS
			SELECT account_id,
				CASE WHEN type='income' THEN amount WHEN type='expense' THEN -amount END AS signed
			FROM transactions
			WHERE type IN ('income','expense')
			UNION ALL
			SELECT from_account_id, -amount FROM transactions WHERE type='transfer'
			UNION ALL
			SELECT to_account_id,  +amount FROM transactions WHERE type='transfer';
`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}
