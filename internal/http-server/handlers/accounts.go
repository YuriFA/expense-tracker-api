package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage"

	"github.com/gin-gonic/gin"
)

type AccountRequest struct {
	Name           string   `json:"name"           binding:"required"`
	OpeningBalance *float64 `json:"openingBalance" binding:"required"`
}

type UpdateAccountRequest struct {
	Name             *string  `json:"name"             binding:"omitempty,min=3"`
	ManualAdjustment *float64 `json:"manualAdjustment" binding:"omitempty"`
}

func (h *Handler) CreateAccount(c *gin.Context) {
	op := "handlers.accounts.CreateAccount"

	log := h.Logger.With(
		slog.String("op", op),
	)

	var req AccountRequest
	if !bindAndValidateJSON(c, log, &req) {
		return
	}

	account, err := h.DB.CreateAccount(req.Name, *req.OpeningBalance)
	if err != nil {
		log.Error("failed to create account", logger.Error(err))
		writeError(c, http.StatusInternalServerError, ErrCodeInternal, "failed to create account")
		return
	}

	c.JSON(http.StatusCreated, account)
}

func (h *Handler) UpdateAccount(c *gin.Context) {
	op := "handlers.accounts.UpdateAccount"

	log := h.Logger.With(
		slog.String("op", op),
	)

	var req UpdateAccountRequest
	if !bindAndValidateJSON(c, log, &req) {
		return
	}

	if req.Name == nil && req.ManualAdjustment == nil {
		writeError(c, http.StatusBadRequest, ErrCodeValidationFailed, "no fields to update")
		return
	}

	id := c.Param("id")
	account, err := h.DB.UpdateAccount(id, storage.UpdateAccountParams{
		Name:             req.Name,
		ManualAdjustment: req.ManualAdjustment,
	})
	if err != nil {
		if errors.Is(err, storage.ErrAccountNotFound) {
			log.Info("account not found", slog.String("id", id))
			writeError(c, http.StatusNotFound, ErrCodeAccountNotFound, "account not found")
			return
		}

		log.Error("failed to update account", logger.Error(err))
		writeError(c, http.StatusInternalServerError, ErrCodeInternal, "failed to update account")
		return
	}

	c.JSON(http.StatusOK, account)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	op := "handlers.accounts.DeleteAccount"

	log := h.Logger.With(
		slog.String("op", op),
	)

	id := c.Param("id")
	err := h.DB.DeleteAccount(id)
	if err != nil {
		if errors.Is(err, storage.ErrAccountNotFound) {
			log.Info("account not found", slog.String("id", id))
			writeError(c, http.StatusNotFound, ErrCodeAccountNotFound, "account not found")
			return
		}

		if errors.Is(err, storage.ErrAccountHasTransactions) {
			log.Info("account in use", slog.String("id", id))
			writeError(c, http.StatusConflict, ErrCodeAccountInUse, "account in use")
			return
		}

		log.Error("failed to delete account", logger.Error(err))
		writeError(c, http.StatusInternalServerError, ErrCodeInternal, "failed to delete account")
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) GetAccount(c *gin.Context) {
	op := "handlers.accounts.GetAccount"

	log := h.Logger.With(
		slog.String("op", op),
	)

	id := c.Param("id")
	account, err := h.DB.GetAccount(id)
	if err != nil {
		if errors.Is(err, storage.ErrAccountNotFound) {
			log.Info("account not found", slog.String("id", id))
			writeError(c, http.StatusNotFound, ErrCodeAccountNotFound, "account not found")
			return
		}

		log.Error("failed to get account", logger.Error(err))
		writeError(c, http.StatusInternalServerError, ErrCodeInternal, "failed to get account")
		return
	}

	c.JSON(http.StatusOK, account)
}

func (h *Handler) ListAccounts(c *gin.Context) {
	op := "handlers.accounts.ListAccounts"

	log := h.Logger.With(
		slog.String("op", op),
	)

	accounts, err := h.DB.GetAccounts()
	if err != nil {
		log.Error("failed to get accounts", logger.Error(err))
		writeError(c, http.StatusInternalServerError, ErrCodeInternal, "failed to get accounts")
		return
	}

	c.JSON(http.StatusOK, accounts)
}

func calculateNetWorth(balances []storage.AccountBalance) float64 {
	var netWorth float64
	for _, balance := range balances {
		netWorth += balance.Balance
	}
	return netWorth
}

func (h *Handler) GetAccountBalances(c *gin.Context) {
	op := "handlers.accounts.GetAccountBalances"
	log := h.Logger.With(
		slog.String("op", op),
	)

	balances, err := h.DB.GetAccountBalances()
	if err != nil {
		log.Error("failed to get account balances", logger.Error(err))
		writeError(
			c,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"failed to get account balances",
		)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"balances": balances,
		"netWorth": calculateNetWorth(balances),
	})
}
