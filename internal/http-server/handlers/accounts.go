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
	const op = "handlers.accounts.CreateAccount"

	log := h.Logger.With(
		slog.String("op", op),
	)

	var req AccountRequest
	if err := c.BindJSON(&req); err != nil {
		log.Error("invalid request body", logger.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accountID, err := h.DB.CreateAccount(req.Name, *req.OpeningBalance)
	if err != nil {
		log.Error("failed to create account", logger.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create account",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Account created successfully",
		"data":    gin.H{"id": accountID},
	})
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	const op = "handlers.accounts.ListAccounts"

	log := h.Logger.With(
		slog.String("op", op),
	)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Error("failed to parse id", logger.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id param",
		})
		return
	}

	err = h.DB.DeleteAccount(id)
	if err != nil {
		log.Error("failed to delete account", logger.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete account",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Delete account",
		"data":    true,
	})
}

func (h *Handler) GetAccount(c *gin.Context) {
	op := "handlers.accounts.GetAccount"

	log := h.Logger.With(
		slog.String("op", op),
	)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Error("failed to parse id", logger.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id param",
		})
		return
	}

	account, err := h.DB.GetAccount(id)
	if err != nil {
		if errors.Is(err, storage.ErrAccountNotFound) {
			log.Info("account not found", slog.Int64("id", id))
			c.JSON(http.StatusNotFound, gin.H{
				"error": "account not found",
			})
			return
		}

		log.Error("failed to get account", logger.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get account",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "account",
		"data":    account,
	})
}

func (h *Handler) ListAccounts(c *gin.Context) {
	const op = "handlers.accounts.ListAccounts"

	log := h.Logger.With(
		slog.String("op", op),
	)

	accounts, err := h.DB.GetAccounts()
	if err != nil {
		log.Error("failed to get accounts", logger.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get accounts",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "List of accounts",
		"data":    accounts,
	})
}
