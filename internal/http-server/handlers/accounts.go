package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/yurifa/expense-tracker-api/internal/http-server/httpctx"
	"github.com/yurifa/expense-tracker-api/internal/http-server/httperr"
	"github.com/yurifa/expense-tracker-api/internal/logger"
	"github.com/yurifa/expense-tracker-api/internal/storage"

	"github.com/gin-gonic/gin"
)

type CreateAccountRequest struct {
	Name           string `json:"name"           binding:"required"`
	Currency       string `json:"currency"       binding:"required"`
	OpeningBalance *int64 `json:"openingBalance" binding:"required"`
}

type UpdateAccountRequest struct {
	Name             *string `json:"name"             binding:"omitempty,min=3"`
	ManualAdjustment *int64  `json:"manualAdjustment" binding:"omitempty"`
}

func (h *Handler) CreateAccount(c *gin.Context) {
	op := "handlers.accounts.CreateAccount"

	log := h.loggerFor(c, op)

	user := httpctx.CurrentUser(c)

	var req CreateAccountRequest
	if !bindAndValidateJSON(c, log, &req) {
		return
	}

	account, err := h.DB.CreateAccount(c.Request.Context(), storage.CreateAccountParams{
		UserID:         user.ID,
		Name:           req.Name,
		Currency:       req.Currency,
		OpeningBalance: *req.OpeningBalance,
	})
	if err != nil {
		log.Error("failed to create account", logger.Error(err))
		httperr.Write(c, http.StatusInternalServerError, httperr.ErrCodeInternal, "failed to create account")
		return
	}

	c.JSON(http.StatusCreated, account)
}

func (h *Handler) UpdateAccount(c *gin.Context) {
	op := "handlers.accounts.UpdateAccount"

	log := h.loggerFor(c, op)

	user := httpctx.CurrentUser(c)

	var req UpdateAccountRequest
	if !bindAndValidateJSON(c, log, &req) {
		return
	}

	if req.Name == nil && req.ManualAdjustment == nil {
		httperr.Write(c, http.StatusBadRequest, httperr.ErrCodeValidationFailed, "no fields to update")
		return
	}

	id := c.Param("id")
	account, err := h.DB.UpdateAccount(c.Request.Context(), user.ID, id, storage.UpdateAccountParams{
		Name:             req.Name,
		ManualAdjustment: req.ManualAdjustment,
	})
	if err != nil {
		if errors.Is(err, storage.ErrAccountNotFound) {
			log.Info("account not found", slog.String("id", id))
			httperr.Write(c, http.StatusNotFound, httperr.ErrCodeAccountNotFound, "account not found")
			return
		}

		log.Error("failed to update account", logger.Error(err))
		httperr.Write(c, http.StatusInternalServerError, httperr.ErrCodeInternal, "failed to update account")
		return
	}

	c.JSON(http.StatusOK, account)
}

//nolint:dupl // CRUD handler intentionally mirrors DeleteCategory structure
func (h *Handler) DeleteAccount(c *gin.Context) {
	op := "handlers.accounts.DeleteAccount"

	log := h.loggerFor(c, op)

	user := httpctx.CurrentUser(c)

	id := c.Param("id")
	err := h.DB.DeleteAccount(c.Request.Context(), user.ID, id)
	if err != nil {
		if errors.Is(err, storage.ErrAccountNotFound) {
			log.Info("account not found", slog.String("id", id))
			httperr.Write(c, http.StatusNotFound, httperr.ErrCodeAccountNotFound, "account not found")
			return
		}

		if errors.Is(err, storage.ErrAccountHasTransactions) {
			log.Info("account in use", slog.String("id", id))
			httperr.Write(c, http.StatusConflict, httperr.ErrCodeAccountInUse, "account in use")
			return
		}

		log.Error("failed to delete account", logger.Error(err))
		httperr.Write(c, http.StatusInternalServerError, httperr.ErrCodeInternal, "failed to delete account")
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) GetAccount(c *gin.Context) {
	op := "handlers.accounts.GetAccount"

	log := h.loggerFor(c, op)

	user := httpctx.CurrentUser(c)

	id := c.Param("id")
	account, err := h.DB.GetAccount(c.Request.Context(), user.ID, id)
	if err != nil {
		if errors.Is(err, storage.ErrAccountNotFound) {
			log.Info("account not found", slog.String("id", id))
			httperr.Write(c, http.StatusNotFound, httperr.ErrCodeAccountNotFound, "account not found")
			return
		}

		log.Error("failed to get account", logger.Error(err))
		httperr.Write(c, http.StatusInternalServerError, httperr.ErrCodeInternal, "failed to get account")
		return
	}

	c.JSON(http.StatusOK, account)
}

func (h *Handler) ListAccounts(c *gin.Context) {
	op := "handlers.accounts.ListAccounts"

	log := h.loggerFor(c, op)

	user := httpctx.CurrentUser(c)

	accounts, err := h.DB.GetAccounts(c.Request.Context(), user.ID)
	if err != nil {
		log.Error("failed to get accounts", logger.Error(err))
		httperr.Write(c, http.StatusInternalServerError, httperr.ErrCodeInternal, "failed to get accounts")
		return
	}

	c.JSON(http.StatusOK, accounts)
}

func calculateNetWorth(balances []storage.AccountBalance) int64 {
	var netWorth int64
	for _, balance := range balances {
		netWorth += balance.Balance
	}
	return netWorth
}

func (h *Handler) GetAccountBalances(c *gin.Context) {
	op := "handlers.accounts.GetAccountBalances"
	log := h.loggerFor(c, op)
	user := httpctx.CurrentUser(c)

	balances, err := h.DB.GetAccountBalances(c.Request.Context(), user.ID)
	if err != nil {
		log.Error("failed to get account balances", logger.Error(err))
		httperr.Write(
			c,
			http.StatusInternalServerError,
			httperr.ErrCodeInternal,
			"failed to get account balances",
		)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"balances": balances,
		"netWorth": calculateNetWorth(balances),
	})
}
