package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"expense-tracker-api/internal/storage"

	"github.com/google/uuid"
)

func (s *Storage) CreateAccount(params storage.CreateAccountParams) (*storage.Account, error) {
	const op = "storage.sqlite.CreateAccount"

	stmt, err := s.db.Prepare(
		`INSERT INTO accounts (id, user_id, name, currency, opening_balance, manual_adjustment)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id, user_id, name, currency, opening_balance, manual_adjustment, opening_balance + manual_adjustment, created_at, updated_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	id := uuid.NewString()
	var account storage.Account
	err = stmt.QueryRow(id, params.UserID, params.Name, params.Currency, params.OpeningBalance, int64(0)).
		Scan(&account.ID, &account.UserID, &account.Name, &account.Currency, &account.OpeningBalance, &account.ManualAdjustment, &account.Balance, &account.CreatedAt, &account.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &account, nil
}

func (s *Storage) UpdateAccount(
	userID string,
	id string,
	params storage.UpdateAccountParams,
) (*storage.Account, error) {
	const op = "storage.sqlite.UpdateAccount"

	setParts, args := newUpdateBuilder().
		addString("name", params.Name).
		addAmount("manual_adjustment", params.ManualAdjustment).
		build(", ")

	args = append(args, id)
	args = append(args, userID)

	query := fmt.Sprintf(
		`UPDATE accounts SET %s
		WHERE id = ? AND user_id = ?
		RETURNING id, user_id, name, currency, opening_balance, manual_adjustment,
		(opening_balance + manual_adjustment +
			COALESCE(
				(SELECT SUM(c.signed) FROM account_contributions c WHERE c.account_id = accounts.id),
			0)
		) AS balance,
		created_at, updated_at`,
		setParts,
	)

	var account storage.Account
	err := s.db.QueryRow(query, args...).
		Scan(&account.ID, &account.UserID, &account.Name, &account.Currency, &account.OpeningBalance, &account.ManualAdjustment, &account.Balance, &account.CreatedAt, &account.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrAccountNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &account, nil
}

func (s *Storage) DeleteAccount(userID string, id string) error {
	const op = "storage.sqlite.DeleteAccount"

	stmt, err := s.db.Prepare(
		`DELETE FROM accounts WHERE id = ? AND user_id = ?`,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(id, userID)
	if err != nil {
		if isFKViolationError(err) {
			return fmt.Errorf("%s: %w", op, storage.ErrAccountHasTransactions)
		}
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

func (s *Storage) GetAccount(userID string, id string) (*storage.Account, error) {
	const op = "storage.sqlite.GetAccount"

	stmt, err := s.db.Prepare(
		`SELECT a.id, a.user_id, a.name, a.currency, a.opening_balance, a.manual_adjustment, 
			a.opening_balance + a.manual_adjustment + COALESCE(SUM(c.signed),0) AS balance,
			a.created_at, a.updated_at
		FROM accounts a
		LEFT JOIN account_contributions c ON c.account_id = a.id
		WHERE a.id = ? AND a.user_id = ?
		GROUP BY a.id, a.user_id, a.name, a.currency, a.opening_balance, a.manual_adjustment, a.created_at, a.updated_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var account storage.Account
	err = stmt.QueryRow(id, userID).
		Scan(&account.ID, &account.UserID, &account.Name, &account.Currency, &account.OpeningBalance, &account.ManualAdjustment, &account.Balance, &account.CreatedAt, &account.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrAccountNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &account, nil
}

func (s *Storage) GetAccounts(userID string) ([]storage.Account, error) {
	const op = "storage.sqlite.GetAccounts"

	stmt, err := s.db.Prepare(
		`SELECT a.id, a.user_id, a.name, a.currency, a.opening_balance, a.manual_adjustment, 
			a.opening_balance + a.manual_adjustment + COALESCE(SUM(c.signed),0) AS balance,
			a.created_at, a.updated_at
		FROM accounts a
		LEFT JOIN account_contributions c ON c.account_id = a.id
		WHERE a.user_id = ?
		GROUP BY a.id, a.user_id, a.name, a.currency, a.opening_balance, a.manual_adjustment, a.created_at, a.updated_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	accounts := []storage.Account{}
	rows, err := stmt.Query(userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		account := storage.Account{}
		err := rows.Scan(
			&account.ID,
			&account.UserID,
			&account.Name,
			&account.Currency,
			&account.OpeningBalance,
			&account.ManualAdjustment,
			&account.Balance,
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

func (s *Storage) GetAccountBalances(userID string) ([]storage.AccountBalance, error) {
	const op = "storage.sqlite.GetAccountBalances"

	stmt, err := s.db.Prepare(
		`SELECT a.id, a.user_id, a.name, a.currency, a.opening_balance + a.manual_adjustment + COALESCE(SUM(c.signed), 0) AS balance
		FROM accounts a
		LEFT JOIN account_contributions c ON c.account_id = a.id
		WHERE a.user_id = ?
		GROUP BY a.id, a.user_id, a.name, a.currency, a.opening_balance, a.manual_adjustment`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	balances := []storage.AccountBalance{}
	rows, err := stmt.Query(userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		b := storage.AccountBalance{}
		if err := rows.Scan(&b.ID, &b.UserID, &b.Name, &b.Currency, &b.Balance); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		balances = append(balances, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return balances, nil
}
