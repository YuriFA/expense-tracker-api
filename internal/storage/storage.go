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

var ErrAccountNotFound = errors.New("account not found")
