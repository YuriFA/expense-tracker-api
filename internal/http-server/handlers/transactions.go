package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/util"

	"github.com/gin-gonic/gin"
)

type TransactionRequest struct {
	Type          string    `json:"type"          binding:"required,oneof=income expense transfer"`
	Amount        float64   `json:"amount"        binding:"required,gt=0"`
	Description   string    `json:"description"   binding:"omitempty"`
	OccurredAt    time.Time `json:"occurredAt"    binding:"required"                               time_format:"2006-01-02T15:04:05Z07:00"`
	AccountId     *string   `json:"accountId"     binding:"omitempty,uuid"`
	CategoryId    *string   `json:"categoryId"    binding:"omitempty,uuid"`
	FromAccountId *string   `json:"fromAccountId" binding:"omitempty,uuid"`
	ToAccountId   *string   `json:"toAccountId"   binding:"omitempty,uuid"`
}

type UpdateTransactionRequest struct {
	Amount        *float64   `json:"amount"        binding:"omitempty,gt=0"`
	Description   *string    `json:"description"   binding:"omitempty"`
	OccurredAt    *time.Time `json:"occurredAt"    binding:"omitempty"      time_format:"2006-01-02T15:04:05Z07:00"`
	AccountId     *string    `json:"accountId"     binding:"omitempty,uuid"`
	CategoryId    *string    `json:"categoryId"    binding:"omitempty,uuid"`
	FromAccountId *string    `json:"fromAccountId" binding:"omitempty,uuid"`
	ToAccountId   *string    `json:"toAccountId"   binding:"omitempty,uuid"`
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

func validateTransactionRequest(req TransactionRequest) []FieldError {
	var errs []FieldError
	switch req.Type {
	case "income", "expense":
		if req.AccountId == nil {
			errs = append(errs, FieldError{
				Field:   "accountId",
				Message: "accountId is required",
			})
		}

		if req.CategoryId == nil {
			errs = append(errs, FieldError{
				Field:   "categoryId",
				Message: "categoryId is required",
			})
		}

		if req.FromAccountId != nil {
			errs = append(errs, FieldError{
				Field:   "fromAccountId",
				Message: "not allowed for income or expense transactions",
			})
		}

		if req.ToAccountId != nil {
			errs = append(errs, FieldError{
				Field:   "toAccountId",
				Message: "not allowed for income or expense transactions",
			})
		}

	case "transfer":
		if req.FromAccountId == nil {
			errs = append(errs, FieldError{
				Field:   "fromAccountId",
				Message: "fromAccountId is required",
			})
		}
		if req.ToAccountId == nil {
			errs = append(errs, FieldError{
				Field:   "toAccountId",
				Message: "toAccountId is required",
			})
		}
		if req.AccountId != nil {
			errs = append(errs, FieldError{
				Field:   "accountId",
				Message: "not allowed for transfer transactions",
			})
		}
		if req.CategoryId != nil {
			errs = append(errs, FieldError{
				Field:   "categoryId",
				Message: "not allowed for transfer transactions",
			})
		}
	}
	return errs
}

type commonTransactionParamsReq struct {
	AccountId     *string
	CategoryId    *string
	FromAccountId *string
	ToAccountId   *string
}

func writeTransactionError(
	c *gin.Context,
	log *slog.Logger,
	err error,
	req commonTransactionParamsReq,
) {
	switch {
	case errors.Is(err, storage.ErrTransactionNotFound):
		log.Info("transaction not found")
		writeError(c, http.StatusNotFound, ErrCodeTransactionNotFound, "transaction not found")
	case errors.Is(err, storage.ErrAccountNotFound):
		log.Info(
			"account not found",
			slog.String("accountId", util.FromPtrOr(req.AccountId, "empty")),
		)
		writeError(
			c,
			http.StatusUnprocessableEntity,
			ErrCodeAccountNotFound,
			"account not found",
		)
	case errors.Is(err, storage.ErrCategoryNotFound):
		log.Info(
			"category not found",
			slog.String("categoryId", util.FromPtrOr(req.CategoryId, "empty")),
		)
		writeError(
			c,
			http.StatusUnprocessableEntity,
			ErrCodeCategoryNotFound,
			"category not found",
		)
	case errors.Is(err, storage.ErrCategoryTypeMismatch):
		log.Info(
			"transaction type does not match category type",
			slog.String(
				"categoryId",
				util.FromPtrOr(req.CategoryId, "empty"),
			),
		)
		writeError(
			c,
			http.StatusUnprocessableEntity,
			ErrCodeCategoryTypeMismatch,
			"transaction type does not match category type",
		)
	case errors.Is(err, storage.ErrSameAccountTransfer):
		log.Info(
			"transaction from and to accounts are the same",
			slog.String(
				"fromAccountId",
				util.FromPtrOr(req.FromAccountId, "empty"),
			),
			slog.String(
				"toAccountId",
				util.FromPtrOr(req.ToAccountId, "empty"),
			),
		)
		writeError(
			c,
			http.StatusUnprocessableEntity,
			ErrCodeSameAccountTransfer,
			"transaction from and to accounts are the same",
		)
	case errors.Is(err, storage.ErrInvalidRefs):
		log.Info(
			"invalid references",
			slog.String(
				"accountId",
				util.FromPtrOr(req.AccountId, "empty"),
			),
			slog.String(
				"categoryId",
				util.FromPtrOr(req.CategoryId, "empty"),
			),
			slog.String(
				"fromAccountId",
				util.FromPtrOr(req.FromAccountId, "empty"),
			),
			slog.String(
				"toAccountId",
				util.FromPtrOr(req.ToAccountId, "empty"),
			),
		)
		writeError(
			c,
			http.StatusUnprocessableEntity,
			ErrCodeInvalidRefs,
			"invalid references",
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

	if errs := validateTransactionRequest(req); len(errs) > 0 {
		c.JSON(http.StatusBadRequest, ValidationErrorResponse{
			ErrorResponse: ErrorResponse{
				Code:    ErrCodeValidationFailed,
				Message: "validation failed",
			},
			Errors: errs,
		})
		return
	}

	transaction, err := h.DB.CreateTransaction(storage.CreateTransactionParams{
		Type:          req.Type,
		Amount:        req.Amount,
		Description:   req.Description,
		OccurredAt:    req.OccurredAt,
		AccountId:     req.AccountId,
		CategoryId:    req.CategoryId,
		FromAccountId: req.FromAccountId,
		ToAccountId:   req.ToAccountId,
	})
	if err != nil {
		log.Info("info", slog.String("type", req.Type))
		writeTransactionError(c, log, err, commonTransactionParamsReq{
			AccountId:     req.AccountId,
			CategoryId:    req.CategoryId,
			FromAccountId: req.FromAccountId,
			ToAccountId:   req.ToAccountId,
		})
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

	if req.AccountId == nil && req.Description == nil &&
		req.OccurredAt == nil && req.CategoryId == nil && req.Amount == nil && req.FromAccountId == nil && req.ToAccountId == nil {
		writeError(c, http.StatusBadRequest, ErrCodeValidationFailed, "no fields to update")
		return
	}

	id := c.Param("id")

	transaction, err := h.DB.UpdateTransaction(id, storage.UpdateTransactionParams{
		Amount:        req.Amount,
		Description:   req.Description,
		OccurredAt:    req.OccurredAt,
		AccountId:     req.AccountId,
		CategoryId:    req.CategoryId,
		FromAccountId: req.FromAccountId,
		ToAccountId:   req.ToAccountId,
	})
	if err != nil {
		log.Info("info", slog.String("id", id))
		writeTransactionError(c, log, err, commonTransactionParamsReq{
			AccountId:     req.AccountId,
			CategoryId:    req.CategoryId,
			FromAccountId: req.FromAccountId,
			ToAccountId:   req.ToAccountId,
		})
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

	if params.ToDate != nil {
		params.ToDate = new(endOfDay(*params.ToDate))
	}

	log.Debug(
		"query parameters after parse",
		slog.Any("params", params),
	)
	transactions, err := h.DB.GetTransactions(storage.GetTransactionsParams{
		Type:       params.Type,
		AccountId:  params.AccountId,
		CategoryId: params.CategoryId,
		FromDate:   params.FromDate,
		ToDate:     params.ToDate,
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
