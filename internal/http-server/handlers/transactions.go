package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/yurifa/expense-tracker-api/internal/http-server/httpctx"
	"github.com/yurifa/expense-tracker-api/internal/http-server/httperr"
	"github.com/yurifa/expense-tracker-api/internal/logger"
	"github.com/yurifa/expense-tracker-api/internal/storage"
	"github.com/yurifa/expense-tracker-api/internal/util"

	"github.com/gin-gonic/gin"
)

type TransactionRequest struct {
	Type          storage.TransactionType `json:"type"          binding:"required,oneof=income expense transfer"`
	Amount        int64                   `json:"amount"        binding:"required,gt=0"`
	Description   string                  `json:"description"   binding:"omitempty"`
	OccurredAt    time.Time               `json:"occurredAt"    binding:"required"                               time_format:"2006-01-02T15:04:05Z07:00"`
	AccountID     *string                 `json:"accountId"     binding:"omitempty,uuid"`
	CategoryID    *string                 `json:"categoryId"    binding:"omitempty,uuid"`
	FromAccountID *string                 `json:"fromAccountId" binding:"omitempty,uuid"`
	ToAccountID   *string                 `json:"toAccountId"   binding:"omitempty,uuid"`
}

type UpdateTransactionRequest struct {
	Amount        *int64     `json:"amount"        binding:"omitempty,gt=0"`
	Description   *string    `json:"description"   binding:"omitempty"`
	OccurredAt    *time.Time `json:"occurredAt"    binding:"omitempty"      time_format:"2006-01-02T15:04:05Z07:00"`
	AccountID     *string    `json:"accountId"     binding:"omitempty,uuid"`
	CategoryID    *string    `json:"categoryId"    binding:"omitempty,uuid"`
	FromAccountID *string    `json:"fromAccountId" binding:"omitempty,uuid"`
	ToAccountID   *string    `json:"toAccountId"   binding:"omitempty,uuid"`
}

type GetTransactionsQuery struct {
	Type       *storage.TransactionType `form:"type"       binding:"omitempty,oneof=income expense transfer"`
	AccountID  *string                  `form:"accountId"  binding:"omitempty,uuid"`
	CategoryID *string                  `form:"categoryId" binding:"omitempty,uuid"`
	FromDate   *time.Time               `form:"fromDate"   binding:"omitempty"                                             time_format:"2006-01-02"`
	ToDate     *time.Time               `form:"toDate"     binding:"omitempty,gtefield=FromDate"                           time_format:"2006-01-02"`
	Limit      *int                     `form:"limit"      binding:"omitempty,gt=0"`
	Sort       *storage.SortParam       `form:"sort"       binding:"omitempty,oneof=occurredAt -occurredAt amount -amount"`
}

func validateTransactionRequest(req TransactionRequest) []httperr.FieldError {
	var errs []httperr.FieldError
	switch req.Type {
	case storage.TransactionTypeIncome, storage.TransactionTypeExpense:
		if req.AccountID == nil {
			errs = append(errs, httperr.FieldError{
				Field:   "accountId",
				Message: "accountId is required",
			})
		}

		if req.CategoryID == nil {
			errs = append(errs, httperr.FieldError{
				Field:   "categoryId",
				Message: "categoryId is required",
			})
		}

		if req.FromAccountID != nil {
			errs = append(errs, httperr.FieldError{
				Field:   "fromAccountId",
				Message: "not allowed for income or expense transactions",
			})
		}

		if req.ToAccountID != nil {
			errs = append(errs, httperr.FieldError{
				Field:   "toAccountId",
				Message: "not allowed for income or expense transactions",
			})
		}

	case storage.TransactionTypeTransfer:
		if req.FromAccountID == nil {
			errs = append(errs, httperr.FieldError{
				Field:   "fromAccountId",
				Message: "fromAccountId is required",
			})
		}
		if req.ToAccountID == nil {
			errs = append(errs, httperr.FieldError{
				Field:   "toAccountId",
				Message: "toAccountId is required",
			})
		}
		if req.AccountID != nil {
			errs = append(errs, httperr.FieldError{
				Field:   "accountId",
				Message: "not allowed for transfer transactions",
			})
		}
		if req.CategoryID != nil {
			errs = append(errs, httperr.FieldError{
				Field:   "categoryId",
				Message: "not allowed for transfer transactions",
			})
		}
	}
	return errs
}

func validateUpdateTransactionRequest(
	currentType storage.TransactionType,
	req UpdateTransactionRequest,
) []httperr.FieldError {
	var errs []httperr.FieldError
	switch currentType {
	case storage.TransactionTypeIncome, storage.TransactionTypeExpense:
		if req.FromAccountID != nil {
			errs = append(errs, httperr.FieldError{
				Field:   "fromAccountId",
				Message: "not allowed for income or expense transactions",
			})
		}
		if req.ToAccountID != nil {
			errs = append(errs, httperr.FieldError{
				Field:   "toAccountId",
				Message: "not allowed for income or expense transactions",
			})
		}
	case storage.TransactionTypeTransfer:
		if req.AccountID != nil {
			errs = append(errs, httperr.FieldError{
				Field:   "accountId",
				Message: "not allowed for transfer transactions",
			})
		}
		if req.CategoryID != nil {
			errs = append(errs, httperr.FieldError{
				Field:   "categoryId",
				Message: "not allowed for transfer transactions",
			})
		}
	}
	return errs
}

type commonTransactionParamsReq struct {
	AccountID     *string
	CategoryID    *string
	FromAccountID *string
	ToAccountID   *string
}

//nolint:funlen // flat switch with 6 cases, single-screen readable
func writeTransactionError(
	c *gin.Context,
	log *slog.Logger,
	err error,
	req commonTransactionParamsReq,
) {
	switch {
	case errors.Is(err, storage.ErrTransactionNotFound):
		log.Info("transaction not found")
		httperr.Write(
			c,
			http.StatusNotFound,
			httperr.ErrCodeTransactionNotFound,
			"transaction not found",
		)
	case errors.Is(err, storage.ErrAccountNotFound):
		log.Info(
			"account not found",
			slog.String("accountId", util.FromPtrOr(req.AccountID, "empty")),
		)
		httperr.Write(
			c,
			http.StatusUnprocessableEntity,
			httperr.ErrCodeAccountNotFound,
			"account not found",
		)
	case errors.Is(err, storage.ErrCategoryNotFound):
		log.Info(
			"category not found",
			slog.String("categoryId", util.FromPtrOr(req.CategoryID, "empty")),
		)
		httperr.Write(
			c,
			http.StatusUnprocessableEntity,
			httperr.ErrCodeCategoryNotFound,
			"category not found",
		)
	case errors.Is(err, storage.ErrCategoryTypeMismatch):
		log.Info(
			"transaction type does not match category type",
			slog.String(
				"categoryId",
				util.FromPtrOr(req.CategoryID, "empty"),
			),
		)
		httperr.Write(
			c,
			http.StatusUnprocessableEntity,
			httperr.ErrCodeCategoryTypeMismatch,
			"transaction type does not match category type",
		)
	case errors.Is(err, storage.ErrSameAccountTransfer):
		log.Info(
			"transaction from and to accounts are the same",
			slog.String(
				"fromAccountId",
				util.FromPtrOr(req.FromAccountID, "empty"),
			),
			slog.String(
				"toAccountId",
				util.FromPtrOr(req.ToAccountID, "empty"),
			),
		)
		httperr.Write(
			c,
			http.StatusUnprocessableEntity,
			httperr.ErrCodeSameAccountTransfer,
			"transaction from and to accounts are the same",
		)
	case errors.Is(err, storage.ErrInvalidRefs):
		log.Info(
			"invalid references",
			slog.String(
				"accountId",
				util.FromPtrOr(req.AccountID, "empty"),
			),
			slog.String(
				"categoryId",
				util.FromPtrOr(req.CategoryID, "empty"),
			),
			slog.String(
				"fromAccountId",
				util.FromPtrOr(req.FromAccountID, "empty"),
			),
			slog.String(
				"toAccountId",
				util.FromPtrOr(req.ToAccountID, "empty"),
			),
		)
		httperr.Write(
			c,
			http.StatusUnprocessableEntity,
			httperr.ErrCodeInvalidRefs,
			"invalid references",
		)
	default:
		log.Error("failed to create transaction", logger.Error(err))
		httperr.Write(
			c,
			http.StatusInternalServerError,
			httperr.ErrCodeInternal,
			"failed to create transaction",
		)
	}
}

func (h *Handler) CreateTransaction(c *gin.Context) {
	op := "handlers.transactions.CreateTransaction"

	log := h.Logger.With(
		slog.String("op", op),
	)

	user := httpctx.CurrentUser(c)

	var req TransactionRequest
	if !bindAndValidateJSON(c, log, &req) {
		return
	}

	if errs := validateTransactionRequest(req); len(errs) > 0 {
		httperr.WriteValidation(c, httperr.ErrCodeValidationFailed, "validation failed", errs)
		return
	}

	transaction, err := h.DB.CreateTransaction(c.Request.Context(), storage.CreateTransactionParams{
		UserID:        user.ID,
		Type:          req.Type,
		Amount:        req.Amount,
		Description:   req.Description,
		OccurredAt:    req.OccurredAt,
		AccountID:     req.AccountID,
		CategoryID:    req.CategoryID,
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
	})
	if err != nil {
		writeTransactionError(c, log, err, commonTransactionParamsReq{
			AccountID:     req.AccountID,
			CategoryID:    req.CategoryID,
			FromAccountID: req.FromAccountID,
			ToAccountID:   req.ToAccountID,
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

	user := httpctx.CurrentUser(c)

	var req UpdateTransactionRequest
	if !bindAndValidateJSON(c, log, &req) {
		return
	}

	if req.AccountID == nil && req.Description == nil &&
		req.OccurredAt == nil && req.CategoryID == nil && req.Amount == nil && req.FromAccountID == nil && req.ToAccountID == nil {
		httperr.Write(
			c,
			http.StatusBadRequest,
			httperr.ErrCodeValidationFailed,
			"no fields to update",
		)
		return
	}

	id := c.Param("id")

	current, err := h.DB.GetTransaction(c.Request.Context(), user.ID, id)
	if err != nil {
		if errors.Is(err, storage.ErrTransactionNotFound) {
			log.Info("transaction not found", slog.String("id", id))
			httperr.Write(
				c,
				http.StatusNotFound,
				httperr.ErrCodeTransactionNotFound,
				"transaction not found",
			)
			return
		}
		log.Error("failed to get transaction", logger.Error(err))
		httperr.Write(
			c,
			http.StatusInternalServerError,
			httperr.ErrCodeInternal,
			"failed to get transaction",
		)
		return
	}

	if errs := validateUpdateTransactionRequest(current.Type, req); len(errs) > 0 {
		httperr.WriteValidation(c, httperr.ErrCodeValidationFailed, "validation failed", errs)
		return
	}

	transaction, err := h.DB.UpdateTransaction(c.Request.Context(), user.ID, id, storage.UpdateTransactionParams{
		Amount:        req.Amount,
		Description:   req.Description,
		OccurredAt:    req.OccurredAt,
		AccountID:     req.AccountID,
		CategoryID:    req.CategoryID,
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
	})
	if err != nil {
		writeTransactionError(c, log, err, commonTransactionParamsReq{
			AccountID:     req.AccountID,
			CategoryID:    req.CategoryID,
			FromAccountID: req.FromAccountID,
			ToAccountID:   req.ToAccountID,
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

	user := httpctx.CurrentUser(c)

	id := c.Param("id")

	err := h.DB.DeleteTransaction(c.Request.Context(), user.ID, id)
	if err != nil {
		if errors.Is(err, storage.ErrTransactionNotFound) {
			log.Info("transaction not found", slog.String("id", id))
			httperr.Write(
				c,
				http.StatusNotFound,
				httperr.ErrCodeTransactionNotFound,
				"transaction not found",
			)
			return
		}

		log.Error("failed to delete transaction", logger.Error(err))
		httperr.Write(
			c,
			http.StatusInternalServerError,
			httperr.ErrCodeInternal,
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

	user := httpctx.CurrentUser(c)

	id := c.Param("id")
	transaction, err := h.DB.GetTransaction(c.Request.Context(), user.ID, id)
	if err != nil {
		if errors.Is(err, storage.ErrTransactionNotFound) {
			log.Info("transaction not found", slog.String("id", id))
			httperr.Write(
				c,
				http.StatusNotFound,
				httperr.ErrCodeTransactionNotFound,
				"transaction not found",
			)
			return
		}

		log.Error("failed to get transaction", logger.Error(err))
		httperr.Write(
			c,
			http.StatusInternalServerError,
			httperr.ErrCodeInternal,
			"failed to get transaction",
		)
		return
	}

	c.JSON(http.StatusOK, transaction)
}

func (h *Handler) ListTransactions(c *gin.Context) {
	op := "handlers.transactions.ListTransactions"

	log := h.Logger.With(
		slog.String("op", op),
	)

	user := httpctx.CurrentUser(c)

	var params GetTransactionsQuery
	if !bindAndValidateQuery(c, log, &params) {
		return
	}

	if params.ToDate != nil {
		params.ToDate = new(util.EndOfDay(*params.ToDate))
	}

	log.Debug(
		"query parameters after parse",
		slog.Any("params", params),
	)
	transactions, err := h.DB.GetTransactions(c.Request.Context(), user.ID, storage.GetTransactionsParams{
		Type:       params.Type,
		AccountID:  params.AccountID,
		CategoryID: params.CategoryID,
		FromDate:   params.FromDate,
		ToDate:     params.ToDate,
		Limit:      params.Limit,
		Sort:       params.Sort,
	})
	if err != nil {
		log.Error("failed to get transactions", logger.Error(err))
		httperr.Write(
			c,
			http.StatusInternalServerError,
			httperr.ErrCodeInternal,
			"failed to get transactions",
		)
		return
	}

	c.JSON(http.StatusOK, transactions)
}
