package storage

import (
	"errors"
	"time"
)

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	PasswordHash string `json:"-"`
}

type RegisterUserParams struct {
	Email        string
	PasswordHash string
}

type Session struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	ExpiresAt string `json:"expiresAt"`
}

type CreateSessionParams struct {
	SessionID string
	UserID    string
	ExpiresAt time.Time
}

type Account struct {
	ID               string `json:"id"`
	UserID           string `json:"userId"`
	Name             string `json:"name"`
	Balance          int64  `json:"balance"`
	Currency         string `json:"currency"`
	OpeningBalance   int64  `json:"openingBalance"`
	ManualAdjustment int64  `json:"manualAdjustment"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type AccountBalance struct {
	ID       string `json:"id"`
	UserID   string `json:"userId"`
	Name     string `json:"name"`
	Balance  int64  `json:"balance"`
	Currency string `json:"currency"`
}

type CreateAccountParams struct {
	UserID         string
	Name           string
	Currency       string
	OpeningBalance int64
}

type UpdateAccountParams struct {
	Name             *string
	ManualAdjustment *int64
}

type Category struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Icon      string `json:"icon"`
	Color     string `json:"color"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type CreateCategoryParams struct {
	UserID string
	Name   string
	Type   string
	Icon   string
	Color  string
}

type UpdateCategoryParams struct {
	Name   *string
	Type   *string
	Icon   *string
	Color  *string
}

type GetCategoriesParams struct {
	Type   *string
}

type Transaction struct {
	ID          string `json:"id"`
	UserID      string `json:"userId"`
	Type        string `json:"type"`
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
	OccurredAt  string `json:"occurredAt"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	// Cashflow fields
	AccountID  *string `json:"accountId,omitempty"`
	CategoryID *string `json:"categoryId,omitempty"`
	// Transfer fields
	FromAccountID *string `json:"fromAccountId,omitempty"`
	ToAccountID   *string `json:"toAccountId,omitempty"`
}

type CreateTransactionParams struct {
	UserID      string
	Type        string
	Amount      int64
	Description string
	OccurredAt  time.Time
	// Cashflow fields
	AccountID  *string
	CategoryID *string
	// Transfer fields
	FromAccountID *string
	ToAccountID   *string
}

type UpdateTransactionParams struct {
	Amount      *int64
	Description *string
	OccurredAt  *time.Time
	// Cashflow fields
	AccountID  *string
	CategoryID *string
	// Transfer fields
	FromAccountID *string
	ToAccountID   *string
}

type SortParam string

const (
	OccurredAtAsc  SortParam = "occurredAt"
	OccurredAtDesc SortParam = "-occurredAt"
	AmountAsc      SortParam = "amount"
	AmountDesc     SortParam = "-amount"
)

type GetTransactionsParams struct {
	Type       *string
	AccountID  *string
	CategoryID *string
	FromDate   *time.Time
	ToDate     *time.Time
	Limit      *int
	Sort       *SortParam
}

var (
	ErrUserNotFound            = errors.New("user not found")
	ErrUserAlreadyExists       = errors.New("user already exists")
	ErrSessionNotFound         = errors.New("session not found")
	ErrSessionExpired          = errors.New("session expired")
	ErrAccountNotFound         = errors.New("account not found")
	ErrCategoryNotFound        = errors.New("category not found")
	ErrCategoryAlreadyExists   = errors.New("category already exists")
	ErrCategoryTypeMismatch    = errors.New("category type mismatch")
	ErrTransactionNotFound     = errors.New("transaction not found")
	ErrUnknownSort             = errors.New("unknown sort")
	ErrAccountHasTransactions  = errors.New("account has transactions and cannot be deleted")
	ErrCategoryHasTransactions = errors.New("category has transactions and cannot be deleted")
	ErrInvalidRefs             = errors.New("invalid references in transaction")
	ErrSameAccountTransfer     = errors.New("transfer cannot be made to the same account")
)
