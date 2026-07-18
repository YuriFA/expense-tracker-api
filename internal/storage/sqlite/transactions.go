package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/yurifa/expense-tracker-api/internal/storage"

	"github.com/google/uuid"
)

type validateTransactionRefsParams struct {
	Type          storage.TransactionType
	UserID        string
	AccountID     *string
	CategoryID    *string
	FromAccountID *string
	ToAccountID   *string
}

func (s *Storage) validateTransactionRefs(
	ctx context.Context,
	params validateTransactionRefsParams,
) error {
	switch params.Type {
	case storage.TransactionTypeIncome, storage.TransactionTypeExpense:
		if params.FromAccountID != nil || params.ToAccountID != nil {
			return storage.ErrInvalidRefs
		}

		if params.AccountID == nil || params.CategoryID == nil {
			return storage.ErrInvalidRefs
		}
		// Business rule: account must exist
		_, err := s.GetAccount(ctx, params.UserID, *params.AccountID)
		if err != nil {
			return err
		}
		// Business rule: category must exist
		category, err := s.GetCategory(ctx, params.UserID, *params.CategoryID)
		if err != nil {
			return err
		}

		// Business rule: transaction type must match category type
		if category.Type != params.Type {
			return storage.ErrCategoryTypeMismatch
		}
	case storage.TransactionTypeTransfer:
		if params.AccountID != nil || params.CategoryID != nil {
			return storage.ErrInvalidRefs
		}

		if params.FromAccountID == nil || params.ToAccountID == nil {
			return storage.ErrInvalidRefs
		}
		// Business rule: from account must exist
		_, err := s.GetAccount(ctx, params.UserID, *params.FromAccountID)
		if err != nil {
			return err
		}
		// Business rule: to account must exist
		_, err = s.GetAccount(ctx, params.UserID, *params.ToAccountID)
		if err != nil {
			return err
		}
		// Business rule: from and to accounts must be different
		if *params.FromAccountID == *params.ToAccountID {
			return storage.ErrSameAccountTransfer
		}
	}

	return nil
}

func (s *Storage) CreateTransaction(ctx context.Context,
	params storage.CreateTransactionParams,
) (*storage.Transaction, error) {
	const op = "storage.sqlite.CreateTransaction"

	if err := s.validateTransactionRefs(ctx, validateTransactionRefsParams{
		Type:          params.Type,
		UserID:        params.UserID,
		AccountID:     params.AccountID,
		CategoryID:    params.CategoryID,
		FromAccountID: params.FromAccountID,
		ToAccountID:   params.ToAccountID,
	}); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err := s.db.PrepareContext(
		ctx,
		`INSERT INTO transactions (id, user_id, type, amount, description, occurred_at, account_id, category_id, from_account_id, to_account_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id, user_id, type, amount, description, occurred_at, created_at, updated_at, version, account_id, category_id, from_account_id, to_account_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	id := uuid.NewString()
	var transaction storage.Transaction
	err = stmt.QueryRowContext(ctx, id, params.UserID, params.Type, params.Amount, params.Description, params.OccurredAt, params.AccountID, params.CategoryID, params.FromAccountID, params.ToAccountID).
		Scan(&transaction.ID, &transaction.UserID, &transaction.Type, &transaction.Amount, &transaction.Description, &transaction.OccurredAt, &transaction.CreatedAt, &transaction.UpdatedAt, &transaction.Version, &transaction.AccountID, &transaction.CategoryID, &transaction.FromAccountID, &transaction.ToAccountID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &transaction, nil
}

func (s *Storage) UpdateTransaction(ctx context.Context,
	userID string,
	id string,
	params storage.UpdateTransactionParams,
) (*storage.Transaction, error) {
	const op = "storage.sqlite.UpdateTransaction"

	current, err := s.GetTransaction(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	effectiveAccountID := current.AccountID
	if params.AccountID != nil {
		effectiveAccountID = params.AccountID
	}
	effectiveCategoryID := current.CategoryID
	if params.CategoryID != nil {
		effectiveCategoryID = params.CategoryID
	}
	effectiveFromAccountID := current.FromAccountID
	if params.FromAccountID != nil {
		effectiveFromAccountID = params.FromAccountID
	}
	effectiveToAccountID := current.ToAccountID
	if params.ToAccountID != nil {
		effectiveToAccountID = params.ToAccountID
	}

	if err := s.validateTransactionRefs(ctx, validateTransactionRefsParams{
		Type:          current.Type,
		UserID:        userID,
		AccountID:     effectiveAccountID,
		CategoryID:    effectiveCategoryID,
		FromAccountID: effectiveFromAccountID,
		ToAccountID:   effectiveToAccountID,
	}); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	setParts, args := newUpdateBuilder().
		addAmount("amount", params.Amount).
		addString("description", params.Description).
		addTimeSet("occurred_at", params.OccurredAt).
		addString("account_id", params.AccountID).
		addString("category_id", params.CategoryID).
		addString("from_account_id", params.FromAccountID).
		addString("to_account_id", params.ToAccountID).
		build(", ")

	// Relative increment; literal is safe — no user input.
	setParts = "version = version + 1, " + setParts

	args = append(args, id)
	args = append(args, userID)
	args = append(args, params.Version)

	query := fmt.Sprintf( //nolint:gosec // G201: setParts is built from a fixed whitelist, not user input
		`UPDATE transactions SET %s WHERE id = ? AND user_id = ? AND version = ? RETURNING id, user_id, type, amount, description, occurred_at, created_at, updated_at, version, account_id, category_id, from_account_id, to_account_id`,
		setParts,
	)

	var transaction storage.Transaction
	err = s.db.QueryRowContext(ctx, query, args...).
		Scan(&transaction.ID, &transaction.UserID, &transaction.Type, &transaction.Amount, &transaction.Description, &transaction.OccurredAt, &transaction.CreatedAt, &transaction.UpdatedAt, &transaction.Version, &transaction.AccountID, &transaction.CategoryID, &transaction.FromAccountID, &transaction.ToAccountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// GetTransaction above verified the row exists; if RETURNING
			// returns no rows here, version mismatch is the cause. Edge case:
			// row deleted between Get and Update also surfaces as 409 here,
			// acceptable for this scope.
			return nil, fmt.Errorf("%s: %w", op, storage.ErrTransactionVersionConflict)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &transaction, nil
}

func (s *Storage) DeleteTransaction(ctx context.Context, userID string, id string) error {
	const op = "storage.sqlite.DeleteTransaction"

	stmt, err := s.db.PrepareContext(ctx, `DELETE FROM transactions WHERE id = ? AND user_id = ?`)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrTransactionNotFound)
	}

	return nil
}

func (s *Storage) GetTransaction(
	ctx context.Context,
	userID string,
	id string,
) (*storage.Transaction, error) {
	const op = "storage.sqlite.GetTransaction"

	stmt, err := s.db.PrepareContext(
		ctx,
		`SELECT id, user_id, type, amount, description, occurred_at, created_at, updated_at, version, account_id, category_id, from_account_id, to_account_id FROM transactions WHERE id = ? AND user_id = ?`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var transaction storage.Transaction
	err = stmt.QueryRowContext(ctx, id, userID).
		Scan(&transaction.ID, &transaction.UserID, &transaction.Type, &transaction.Amount, &transaction.Description, &transaction.OccurredAt, &transaction.CreatedAt, &transaction.UpdatedAt, &transaction.Version, &transaction.AccountID, &transaction.CategoryID, &transaction.FromAccountID, &transaction.ToAccountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrTransactionNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &transaction, nil
}

func (s *Storage) GetTransactions(ctx context.Context,
	userID string,
	params storage.GetTransactionsParams,
) ([]storage.Transaction, error) {
	const op = "storage.sqlite.GetTransactions"

	whereParts, args := newWhereBuilder().
		addString("user_id", &userID).
		addTransactionType("type", params.Type).
		addStringsForOr([]string{"account_id", "from_account_id", "to_account_id"}, params.AccountID).
		addString("category_id", params.CategoryID).
		addTimeOp("occurred_at", params.FromDate, ">=").
		addTimeOp("occurred_at", params.ToDate, "<=").
		build(" AND ")

	query := "SELECT id, user_id, type, amount, description, occurred_at, created_at, updated_at, version, account_id, category_id, from_account_id, to_account_id FROM transactions"
	if len(whereParts) > 0 {
		query = fmt.Sprintf(`%s WHERE %s`, query, whereParts)
	}

	allowedSorts := map[storage.SortParam]string{
		storage.OccurredAtAsc:  "occurred_at ASC",
		storage.OccurredAtDesc: "occurred_at DESC",
		storage.AmountAsc:      "amount ASC",
		storage.AmountDesc:     "amount DESC",
	}
	sortParam := storage.OccurredAtDesc
	if params.Sort != nil {
		sortParam = *params.Sort
	}
	sqlSort, ok := allowedSorts[sortParam]
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, storage.ErrUnknownSort)
	}
	query = fmt.Sprintf(`%s ORDER BY %s`, query, sqlSort)

	if params.Limit != nil {
		query = fmt.Sprintf(`%s LIMIT ?`, query)
		args = append(args, *params.Limit)
	}

	transactions := []storage.Transaction{}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		transaction := storage.Transaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.UserID,
			&transaction.Type,
			&transaction.Amount,
			&transaction.Description,
			&transaction.OccurredAt,
			&transaction.CreatedAt,
			&transaction.UpdatedAt,
			&transaction.Version,
			&transaction.AccountID,
			&transaction.CategoryID,
			&transaction.FromAccountID,
			&transaction.ToAccountID,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return transactions, nil
}
