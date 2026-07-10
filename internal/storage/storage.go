package storage

import (
	"errors"
	"time"
)

type User struct {
	Id           string `json:"id"`
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
	Id               string `json:"id"`
	Name             string `json:"name"`
	Balance          int64  `json:"balance"`
	Currency         string `json:"currency"`
	OpeningBalance   int64  `json:"openingBalance"`
	ManualAdjustment int64  `json:"manualAdjustment"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type AccountBalance struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Balance  int64  `json:"balance"`
	Currency string `json:"currency"`
}

type CreateAccountParams struct {
	Name           string
	Currency       string
	OpeningBalance int64
}

type UpdateAccountParams struct {
	Name             *string
	ManualAdjustment *int64
}

type Category struct {
	Id        string `json:"id"`
	UserId    string `json:"userId"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Icon      string `json:"icon"`
	Color     string `json:"color"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type CreateCategoryParams struct {
	UserId string
	Name   string
	Type   string
	Icon   string
	Color  string
}

type UpdateCategoryParams struct {
	Name  *string
	Type  *string
	Icon  *string
	Color *string
}

type GetCategoriesParams struct {
	Type *string
}

type Transaction struct {
	Id          string `json:"id"`
	Type        string `json:"type"`
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
	OccurredAt  string `json:"occurredAt"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	// Cashflow fields
	AccountId  *string `json:"accountId,omitempty"`
	CategoryId *string `json:"categoryId,omitempty"`
	// Transfer fields
	FromAccountId *string `json:"fromAccountId,omitempty"`
	ToAccountId   *string `json:"toAccountId,omitempty"`
}

type CreateTransactionParams struct {
	Type        string
	Amount      int64
	Description string
	OccurredAt  time.Time
	// Cashflow fields
	AccountId  *string
	CategoryId *string
	// Transfer fields
	FromAccountId *string
	ToAccountId   *string
}

type UpdateTransactionParams struct {
	Amount      *int64
	Description *string
	OccurredAt  *time.Time
	// Cashflow fields
	AccountId  *string
	CategoryId *string
	// Transfer fields
	FromAccountId *string
	ToAccountId   *string
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
	AccountId  *string
	CategoryId *string
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
	ErrCategoryTypeMismatch    = errors.New("category type mismatch")
	ErrTransactionNotFound     = errors.New("transaction not found")
	ErrUnknownSort             = errors.New("unknown sort")
	ErrAccountHasTransactions  = errors.New("account has transactions and cannot be deleted")
	ErrCategoryHasTransactions = errors.New("category has transactions and cannot be deleted")
	ErrInvalidRefs             = errors.New("invalid references in transaction")
	ErrSameAccountTransfer     = errors.New("transfer cannot be made to the same account")
)
