package storage

import "errors"

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
	Id        string `json:"id"`
	Name      string `json:"name"`
	Slug      *string `json:"slug"`
	Type      string `json:"type"`
	Icon      string `json:"icon"`
	Color     string `json:"color"`
	IsDefault bool   `json:"isDefault"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type CreateCategoryParams struct {
	Id        string
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

var (
	ErrAccountNotFound  = errors.New("account not found")
	ErrCategoryNotFound = errors.New("category not found")
)
