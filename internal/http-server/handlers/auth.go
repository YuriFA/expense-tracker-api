package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"expense-tracker-api/internal/auth"
	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage"

	"github.com/gin-gonic/gin"
)

type RegisterUserParams struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

func (h *Handler) Register(c *gin.Context) {
	op := "handlers.auth.Register"

	log := h.Logger.With(
		slog.String("op", op),
	)

	var req RegisterUserParams
	if !bindAndValidateJSON(c, log, &req) {
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Error("failed to hash password", logger.Error(err))
		writeError(c, http.StatusInternalServerError, ErrCodeInternal, "failed to hash password")
		return
	}

	user, err := h.DB.RegisterUser(storage.RegisterUserParams{
		Email:        req.Email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		if errors.Is(err, storage.ErrUserAlreadyExists) {
			log.Info("user already exists", logger.Error(err))
			writeError(c, http.StatusConflict, ErrCodeUserAlreadyExists, "user already exists")
			return
		}
		log.Error("failed to register user", logger.Error(err))
		writeError(c, http.StatusInternalServerError, ErrCodeInternal, "failed to register user")
		return
	}

	c.JSON(http.StatusCreated, user)
}
