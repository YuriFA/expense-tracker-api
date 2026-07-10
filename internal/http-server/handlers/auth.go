package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"expense-tracker-api/internal/auth"
	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage"

	"github.com/gin-gonic/gin"
)

type RegisterUserParams struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type LoginUserParams struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
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

func (h *Handler) Login(c *gin.Context) {
	op := "handlers.auth.Login"

	log := h.Logger.With(
		slog.String("op", op),
	)

	var req LoginUserParams
	if !bindAndValidateJSON(c, log, &req) {
		return
	}

	user, err := h.DB.GetUserByEmail(req.Email)
	if err != nil {
		log.Error("failed to get user by email", logger.Error(err))
		writeError(
			c,
			http.StatusUnauthorized,
			ErrCodeInvalidCredentials,
			"invalid credentials",
		)
		return
	}
	err = auth.VerifyPassword(user.PasswordHash, req.Password)
	if err != nil {
		log.Info("invalid credentials", logger.Error(err))
		writeError(c, http.StatusUnauthorized, ErrCodeInvalidCredentials, "invalid credentials")
		return
	}

	sessionID, err := auth.GenerateSessionToken()
	if err != nil {
		log.Error("failed to generate session token", logger.Error(err))
		writeError(
			c,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"failed to generate session token",
		)
		return
	}

	session, err := h.DB.CreateSession(storage.CreateSessionParams{
		SessionID: sessionID,
		UserID:    user.Id,
		ExpiresAt: time.Now().UTC().Add(h.Config.SessionConfig.TTL),
	})
	if err != nil {
		log.Error("failed to login", logger.Error(err))
		writeError(c, http.StatusInternalServerError, ErrCodeInternal, "failed to login")
		return
	}

	c.SetCookieData(
		&http.Cookie{
			Name:     h.Config.SessionConfig.CookieName,
			Value:    session.ID,
			MaxAge:   int(h.Config.SessionConfig.TTL.Seconds()),
			Path:     "/",
			Domain:   "",
			Secure:   h.Config.SessionConfig.Secure,
			HttpOnly: true,
			SameSite: parseSameSite(h.Config.SessionConfig.SameSite),
		},
	)

	c.JSON(http.StatusOK, user)
}
