package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"expense-tracker-api/internal/storage"

	"github.com/google/uuid"
)

type validateTransactionRefsParams struct {
	Type          string
	AccountId     *string
	CategoryId    *string
	FromAccountId *string
	ToAccountId   *string
}

func (s *Storage) validateTransactionRefs(params validateTransactionRefsParams) error {
	switch params.Type {
	case "income", "expense":
		if params.FromAccountId != nil || params.ToAccountId != nil {
			return storage.ErrInvalidRefs
		}

		if params.AccountId == nil || params.CategoryId == nil {
			return storage.ErrInvalidRefs
		}
		// Business rule: account must exist
		_, err := s.GetAccount(*params.AccountId)
		if err != nil {
			return err
		}
		// Business rule: category must exist
		category, err := s.GetCategory(*params.CategoryId)
		if err != nil {
			return err
		}

		// Business rule: transaction type must match category type
		if category.Type != params.Type {
			return storage.ErrCategoryTypeMismatch
		}
	case "transfer":
		if params.AccountId != nil || params.CategoryId != nil {
			return storage.ErrInvalidRefs
		}

		if params.FromAccountId == nil || params.ToAccountId == nil {
			return storage.ErrInvalidRefs
		}
		// Business rule: from account must exist
		_, err := s.GetAccount(*params.FromAccountId)
		if err != nil {
			return err
		}
		// Business rule: to account must exist
		_, err = s.GetAccount(*params.ToAccountId)
		if err != nil {
			return err
		}
		// Business rule: from and to accounts must be different
		if *params.FromAccountId == *params.ToAccountId {
			return storage.ErrSameAccountTransfer
		}
	}

	return nil
}

func (s *Storage) CreateTransaction(
	params storage.CreateTransactionParams,
) (*storage.Transaction, error) {
	const op = "storage.sqlite.CreateTransaction"

	if err := s.validateTransactionRefs(validateTransactionRefsParams{
		Type:          params.Type,
		AccountId:     params.AccountId,
		CategoryId:    params.CategoryId,
		FromAccountId: params.FromAccountId,
		ToAccountId:   params.ToAccountId,
	}); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err := s.db.Prepare(
		`INSERT INTO transactions (id, type, amount, description, occurred_at, account_id, category_id, from_account_id, to_account_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id, type, amount, description, occurred_at, created_at, updated_at, account_id, category_id, from_account_id, to_account_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	id := uuid.NewString()
	var transaction storage.Transaction
	err = stmt.QueryRow(id, params.Type, params.Amount, params.Description, params.OccurredAt, params.AccountId, params.CategoryId, params.FromAccountId, params.ToAccountId).
		Scan(&transaction.Id, &transaction.Type, &transaction.Amount, &transaction.Description, &transaction.OccurredAt, &transaction.CreatedAt, &transaction.UpdatedAt, &transaction.AccountId, &transaction.CategoryId, &transaction.FromAccountId, &transaction.ToAccountId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &transaction, nil
}

func (s *Storage) UpdateTransaction(
	id string,
	params storage.UpdateTransactionParams,
) (*storage.Transaction, error) {
	const op = "storage.sqlite.UpdateTransaction"

	current, err := s.GetTransaction(id)
	if err != nil {
		return nil, err
	}

	effectiveAccountId := current.AccountId
	if params.AccountId != nil {
		effectiveAccountId = params.AccountId
	}
	effectiveCategoryId := current.CategoryId
	if params.CategoryId != nil {
		effectiveCategoryId = params.CategoryId
	}
	effectiveFromAccountId := current.FromAccountId
	if params.FromAccountId != nil {
		effectiveFromAccountId = params.FromAccountId
	}
	effectiveToAccountId := current.ToAccountId
	if params.ToAccountId != nil {
		effectiveToAccountId = params.ToAccountId
	}

	if err := s.validateTransactionRefs(validateTransactionRefsParams{
		Type:          current.Type,
		AccountId:     effectiveAccountId,
		CategoryId:    effectiveCategoryId,
		FromAccountId: effectiveFromAccountId,
		ToAccountId:   effectiveToAccountId,
	}); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	setParts, args := newUpdateBuilder().
		addAmount("amount", params.Amount).
		addString("description", params.Description).
		addTimeSet("occurred_at", params.OccurredAt).
		addString("account_id", params.AccountId).
		addString("category_id", params.CategoryId).
		addString("from_account_id", params.FromAccountId).
		addString("to_account_id", params.ToAccountId).
		build(", ")

	args = append(args, id)

	query := fmt.Sprintf(
		`UPDATE transactions SET %s WHERE id = ? RETURNING id, type, amount, description, occurred_at, created_at, updated_at, account_id, category_id, from_account_id, to_account_id`,
		setParts,
	)

	var transaction storage.Transaction
	err = s.db.QueryRow(query, args...).
		Scan(&transaction.Id, &transaction.Type, &transaction.Amount, &transaction.Description, &transaction.OccurredAt, &transaction.CreatedAt, &transaction.UpdatedAt, &transaction.AccountId, &transaction.CategoryId, &transaction.FromAccountId, &transaction.ToAccountId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrTransactionNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &transaction, nil
}

func (s *Storage) DeleteTransaction(id string) error {
	const op = "storage.sqlite.DeleteTransaction"

	stmt, err := s.db.Prepare(
		`DELETE FROM transactions WHERE id = ?`,
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
		return fmt.Errorf("%s: %w", op, storage.ErrTransactionNotFound)
	}

	return nil
}

func (s *Storage) GetTransaction(id string) (*storage.Transaction, error) {
	const op = "storage.sqlite.GetTransaction"

	stmt, err := s.db.Prepare(
		`SELECT id, type, amount, description, occurred_at, created_at, updated_at, account_id, category_id, from_account_id, to_account_id FROM transactions WHERE id = ?`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var transaction storage.Transaction
	err = stmt.QueryRow(id).
		Scan(&transaction.Id, &transaction.Type, &transaction.Amount, &transaction.Description, &transaction.OccurredAt, &transaction.CreatedAt, &transaction.UpdatedAt, &transaction.AccountId, &transaction.CategoryId, &transaction.FromAccountId, &transaction.ToAccountId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrTransactionNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &transaction, nil
}

func (s *Storage) GetTransactions(
	params storage.GetTransactionsParams,
) ([]storage.Transaction, error) {
	const op = "storage.sqlite.GetTransactions"

	whereParts, args := newWhereBuilder().
		addString("type", params.Type).
		addStringsForOr([]string{"account_id", "from_account_id", "to_account_id"}, params.AccountId).
		addString("category_id", params.CategoryId).
		addTimeOp("occurred_at", params.FromDate, ">=").
		addTimeOp("occurred_at", params.ToDate, "<=").
		build(" AND ")

	query := "SELECT id, type, amount, description, occurred_at, created_at, updated_at, account_id, category_id, from_account_id, to_account_id FROM transactions"
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
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		transaction := storage.Transaction{}
		err := rows.Scan(
			&transaction.Id,
			&transaction.Type,
			&transaction.Amount,
			&transaction.Description,
			&transaction.OccurredAt,
			&transaction.CreatedAt,
			&transaction.UpdatedAt,
			&transaction.AccountId,
			&transaction.CategoryId,
			&transaction.FromAccountId,
			&transaction.ToAccountId,
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
