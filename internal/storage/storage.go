package storage

import (
	"errors"
	"time"
)

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

type Category struct {
	Id        string  `json:"id"`
	Name      string  `json:"name"`
	Slug      *string `json:"slug"`
	Type      string  `json:"type"`
	Icon      string  `json:"icon"`
	Color     string  `json:"color"`
	IsDefault bool    `json:"isDefault"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
}

type CreateCategoryParams struct {
	Name      string
	Type      string
	Icon      string
	Color     string
	IsDefault bool
}

type CreateDefaultCategoryParams struct {
	Id        string
	Name      string
	Slug      string
	Type      string
	Icon      string
	Color     string
	IsDefault bool
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
	Id          string  `json:"id"`
	Type        string  `json:"type"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	OccurredAt  string  `json:"occurredAt"`
	AccountId   string  `json:"accountId"`
	CategoryId  string  `json:"categoryId"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

type CreateTransactionParams struct {
	Type        string
	Amount      float64
	Description string
	OccurredAt  string
	AccountId   string
	CategoryId  string
}

type UpdateTransactionParams struct {
	Type        *string
	Amount      *float64
	Description *string
	OccurredAt  *string
	AccountId   *string
	CategoryId  *string
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
	ErrAccountNotFound      = errors.New("account not found")
	ErrCategoryNotFound     = errors.New("category not found")
	ErrCategoryTypeMismatch = errors.New("category type mismatch")
	ErrTransactionNotFound  = errors.New("transaction not found")
	ErrUnknownSort          = errors.New("unknown sort")
)
