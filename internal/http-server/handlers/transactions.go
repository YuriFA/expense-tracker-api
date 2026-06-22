package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage"

	"github.com/gin-gonic/gin"
)

type TransactionRequest struct {
	Type        string  `json:"type"        binding:"required,oneof=income expense transfer"`
	Amount      float64 `json:"amount"      binding:"required,gt=0"`
	Description string  `json:"description" binding:"omitempty"`
	OccurredAt  string  `json:"occurredAt"  binding:"required,datetime=2006-01-02T15:04:05Z07:00"`
	AccountId   string  `json:"accountId"   binding:"required,uuid"`
	CategoryId  string  `json:"categoryId"  binding:"required,uuid"`
}

type UpdateTransactionRequest struct {
	Type        *string  `json:"type"        binding:"omitempty,oneof=income expense transfer"`
	Amount      *float64 `json:"amount"      binding:"omitempty,gt=0"`
	Description *string  `json:"description" binding:"omitempty"`
	OccurredAt  *string  `json:"occurredAt"  binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	AccountId   *string  `json:"accountId"   binding:"omitempty,uuid"`
	CategoryId  *string  `json:"categoryId"  binding:"omitempty,uuid"`
}

type GetTransactionsQuery struct {
	Type       *string            `form:"type"       binding:"omitempty,oneof=income expense transfer"`
	AccountId  *string            `form:"accountId"  binding:"omitempty,uuid"`
	CategoryId *string            `form:"categoryId" binding:"omitempty,uuid"`
	FromDate   *time.Time         `form:"fromDate"   binding:"omitempty"                                             time_format:"2006-01-02"`
	ToDate     *time.Time         `form:"toDate"     binding:"omitempty,gtefield=FromDate"                           time_format:"2006-01-02"`
	Limit      *int               `form:"limit"      binding:"omitempty,gt=0"`
	Sort       *storage.SortParam `form:"sort"       binding:"omitempty,oneof=occurredAt -occurredAt amount -amount"`
}

func (h *Handler) CreateTransaction(c *gin.Context) {
	op := "handlers.transactions.CreateTransaction"

	log := h.Logger.With(
		slog.String("op", op),
	)

	var req TransactionRequest
	if !bindAndValidateJSON(c, log, &req) {
		return
	}

	transaction, err := h.DB.CreateTransaction(storage.CreateTransactionParams{
		Type:        req.Type,
		Amount:      req.Amount,
		Description: req.Description,
		OccurredAt:  req.OccurredAt,
		AccountId:   req.AccountId,
		CategoryId:  req.CategoryId,
	})
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrAccountNotFound):
			log.Info("account not found", slog.String("accountId", req.AccountId))
			writeError(
				c,
				http.StatusUnprocessableEntity,
				ErrCodeAccountNotFound,
				"account not found",
			)
		case errors.Is(err, storage.ErrCategoryNotFound):
			log.Info("category not found", slog.String("categoryId", req.CategoryId))
			writeError(
				c,
				http.StatusUnprocessableEntity,
				ErrCodeCategoryNotFound,
				"category not found",
			)
		case errors.Is(err, storage.ErrCategoryTypeMismatch):
			log.Info(
				"transaction type does not match category type",
				slog.String("categoryId", req.CategoryId),
				slog.String("transactionType", req.Type),
			)
			writeError(
				c,
				http.StatusUnprocessableEntity,
				ErrCodeCategoryTypeMismatch,
				"transaction type does not match category type",
			)
		default:
			log.Error("failed to create transaction", logger.Error(err))
			writeError(
				c,
				http.StatusInternalServerError,
				ErrCodeInternal,
				"failed to create transaction",
			)
		}
		return
	}

	c.JSON(http.StatusCreated, transaction)
}

func (h *Handler) UpdateTransaction(c *gin.Context) {
	op := "handlers.transactions.UpdateTransaction"

	log := h.Logger.With(
		slog.String("op", op),
	)

	var req UpdateTransactionRequest
	if !bindAndValidateJSON(c, log, &req) {
		return
	}

	if req.AccountId == nil && req.Type == nil && req.Description == nil &&
		req.OccurredAt == nil && req.CategoryId == nil && req.Amount == nil {
		writeError(c, http.StatusBadRequest, ErrCodeValidationFailed, "no fields to update")
		return
	}

	id := c.Param("id")

	transaction, err := h.DB.UpdateTransaction(id, storage.UpdateTransactionParams{
		Type:        req.Type,
		Amount:      req.Amount,
		Description: req.Description,
		OccurredAt:  req.OccurredAt,
		AccountId:   req.AccountId,
		CategoryId:  req.CategoryId,
	})
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrAccountNotFound):
			log.Info("account not found")
			writeError(
				c,
				http.StatusUnprocessableEntity,
				ErrCodeAccountNotFound,
				"account not found",
			)
		case errors.Is(err, storage.ErrCategoryNotFound):
			log.Info("category not found")
			writeError(
				c,
				http.StatusUnprocessableEntity,
				ErrCodeCategoryNotFound,
				"category not found",
			)
		case errors.Is(err, storage.ErrCategoryTypeMismatch):
			log.Info("transaction type does not match category type")
			writeError(
				c,
				http.StatusUnprocessableEntity,
				ErrCodeCategoryTypeMismatch,
				"transaction type does not match category type",
			)
		case errors.Is(err, storage.ErrTransactionNotFound):
			log.Info("transaction not found", slog.String("id", id))
			writeError(c, http.StatusNotFound, ErrCodeTransactionNotFound, "transaction not found")
		default:
			log.Error("failed to update transaction", logger.Error(err))
			writeError(
				c,
				http.StatusInternalServerError,
				ErrCodeInternal,
				"failed to update transaction",
			)
		}

		return
	}

	c.JSON(http.StatusOK, transaction)
}

func (h *Handler) DeleteTransaction(c *gin.Context) {
	op := "handlers.transactions.DeleteTransaction"

	log := h.Logger.With(
		slog.String("op", op),
	)

	id := c.Param("id")

	err := h.DB.DeleteTransaction(id)
	if err != nil {
		if errors.Is(err, storage.ErrTransactionNotFound) {
			log.Info("transaction not found", slog.String("id", id))
			writeError(c, http.StatusNotFound, ErrCodeTransactionNotFound, "transaction not found")
			return
		}

		log.Error("failed to delete transaction", logger.Error(err))
		writeError(
			c,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"failed to delete transaction",
		)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) GetTransaction(c *gin.Context) {
	op := "handlers.transactions.GetTransaction"

	log := h.Logger.With(
		slog.String("op", op),
	)

	id := c.Param("id")
	transaction, err := h.DB.GetTransaction(id)
	if err != nil {
		if errors.Is(err, storage.ErrTransactionNotFound) {
			log.Info("transaction not found", slog.String("id", id))
			writeError(c, http.StatusNotFound, ErrCodeTransactionNotFound, "transaction not found")
			return
		}

		log.Error("failed to get transaction", logger.Error(err))
		writeError(c, http.StatusInternalServerError, ErrCodeInternal, "failed to get transaction")
		return
	}

	c.JSON(http.StatusOK, transaction)
}

func (h *Handler) ListTransactions(c *gin.Context) {
	op := "handlers.transactions.ListTransactions"

	log := h.Logger.With(
		slog.String("op", op),
	)

	var params GetTransactionsQuery
	if !bindAndValidateQuery(c, log, &params) {
		return
	}

	// fromDate, toDate, err := parseDateRange(params.FromDate, params.ToDate)
	// if err != nil {
	// 	log.Info("invalid date range", logger.Error(err))
	// 	writeError(c, http.StatusBadRequest, ErrCodeValidationFailed, "invalid date range")
	// 	return
	// }
	log.Debug(
		"query parameters after parse",
		slog.Any("params", params),
		// slog.Any("fromDateRFC3339", fromDate),
		// slog.Any("toDateRFC3339", toDate),
	)
	transactions, err := h.DB.GetTransactions(storage.GetTransactionsParams{
		Type:       params.Type,
		AccountId:  params.AccountId,
		CategoryId: params.CategoryId,
		FromDate:   params.FromDate,
		ToDate:     endOfDay(params.ToDate),
		Limit:      params.Limit,
		Sort:       params.Sort,
	})
	if err != nil {
		log.Error("failed to get transactions", logger.Error(err))
		writeError(c, http.StatusInternalServerError, ErrCodeInternal, "failed to get transactions")
		return
	}

	c.JSON(http.StatusOK, transactions)
}
