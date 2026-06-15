package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AccountRequest struct {
	Name           string   `json:"name"           binding:"required"`
	OpeningBalance *float64 `json:"openingBalance" binding:"required"`
}

type UpdateAccountRequest struct {
	Name             *string  `json:"name"             binding:"omitempty,min=1"`
	ManualAdjustment *float64 `json:"manualAdjustment" binding:"omitempty"`
}

func (h *Handler) CreateAccount(c *gin.Context) {
	op := "handlers.accounts.CreateAccount"

	log := h.Logger.With(
		slog.String("op", op),
	)

	var req AccountRequest
	if err := c.BindJSON(&req); err != nil {
		if verrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
			log.Info("validation failed", logger.Error(err))
			writeValidationError(c, verrs)
			return
		}

		log.Error("invalid request body", logger.Error(err))
		writeError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "invalid request body")
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
	if err := c.BindJSON(&req); err != nil {
		if verrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
			log.Info("validation failed", logger.Error(err))
			writeValidationError(c, verrs)
			return
		}
		log.Error("invalid request body", logger.Error(err))
		writeError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "invalid request body")
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

func (h *Handler) GetAccountBalances(c *gin.Context) {
	op := "handlers.accounts.GetAccountBalances"
	log := h.Logger.With(
		slog.String("op", op),
	)

	log.Info(
		"GetAccountBalances endpoint called, TODO: implement logic to calculate and return account balances",
	)

	c.JSON(http.StatusOK, gin.H{})
}
