package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage"

	"github.com/gin-gonic/gin"
)

type AccountRequest struct {
	Name           string   `json:"name"           binding:"required"`
	OpeningBalance *float64 `json:"openingBalance" binding:"required,gte=0"`
}

func (h *Handler) CreateAccount(c *gin.Context) {
	op := "handlers.accounts.CreateAccount"

	log := h.Logger.With(
		slog.String("op", op),
	)

	var req AccountRequest
	if err := c.BindJSON(&req); err != nil {
		log.Error("invalid request body", logger.Error(err))
		writeError(c, http.StatusBadRequest, ErrCodeValidationFailed, "invalid request body")
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

func (h *Handler) DeleteAccount(c *gin.Context) {
	op := "handlers.accounts.DeleteAccount"

	log := h.Logger.With(
		slog.String("op", op),
	)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Error("invalid id param", logger.Error(err))
		writeError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "invalid id param")
		return
	}

	err = h.DB.DeleteAccount(id)
	if err != nil {
		if errors.Is(err, storage.ErrAccountNotFound) {
			log.Info("account not found", slog.Int64("id", id))
			writeError(c, http.StatusNotFound, ErrCodeAccountNotFound, "account not found")
			return
		}

		log.Error("failed to delete account", logger.Error(err))
		writeError(c, http.StatusInternalServerError, ErrCodeInternal, "failed to delete account")
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (h *Handler) GetAccount(c *gin.Context) {
	op := "handlers.accounts.GetAccount"

	log := h.Logger.With(
		slog.String("op", op),
	)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Error("invalid id param", logger.Error(err))
		writeError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "invalid id param")
		return
	}

	account, err := h.DB.GetAccount(id)
	if err != nil {
		if errors.Is(err, storage.ErrAccountNotFound) {
			log.Info("account not found", slog.Int64("id", id))
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
