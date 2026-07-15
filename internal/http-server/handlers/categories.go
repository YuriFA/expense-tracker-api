package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"expense-tracker-api/internal/http-server/context"
	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/util"

	"github.com/gin-gonic/gin"
)

type CategoryRequest struct {
	Name  string `json:"name"  binding:"required"`
	Type  string `json:"type"  binding:"required,oneof=income expense"`
	Icon  string `json:"icon"  binding:"required"`
	Color string `json:"color" binding:"required"`
}

type UpdateCategoryRequest struct {
	Name  *string `json:"name"  binding:"omitempty,min=1"`
	Type  *string `json:"type"  binding:"omitempty,oneof=income expense"`
	Icon  *string `json:"icon"  binding:"omitempty,min=1"`
	Color *string `json:"color" binding:"omitempty,min=1"`
}

type GetCategoriesQuery struct {
	Type *string `form:"type" binding:"omitempty,oneof=income expense"`
}

func (h *Handler) CreateCategory(c *gin.Context) {
	op := "handlers.categories.CreateCategory"

	log := h.Logger.With(
		slog.String("op", op),
	)

	user := context.CurrentUser(c)

	var req CategoryRequest
	if !bindAndValidateJSON(c, log, &req) {
		return
	}

	category, err := h.DB.CreateCategory(storage.CreateCategoryParams{
		UserID: user.ID,
		Name:   req.Name,
		Type:   req.Type,
		Icon:   req.Icon,
		Color:  req.Color,
	})
	if err != nil {
		if errors.Is(err, storage.ErrCategoryAlreadyExists) {
			log.Info("category duplicate", slog.String("name", req.Name))
			writeError(
				c,
				http.StatusConflict,
				ErrCodeCategoryAlreadyExists,
				"category already exists",
			)
			return
		}

		log.Error("failed to create category", logger.Error(err))
		writeError(c, http.StatusInternalServerError, ErrCodeInternal, "failed to create category")
		return
	}

	c.JSON(http.StatusCreated, category)
}

func (h *Handler) UpdateCategory(c *gin.Context) {
	op := "handlers.categories.UpdateCategory"

	log := h.Logger.With(
		slog.String("op", op),
	)

	user := context.CurrentUser(c)

	var req UpdateCategoryRequest
	if !bindAndValidateJSON(c, log, &req) {
		return
	}

	if req.Name == nil && req.Type == nil && req.Icon == nil &&
		req.Color == nil {
		writeError(c, http.StatusBadRequest, ErrCodeValidationFailed, "no fields to update")
		return
	}

	id := c.Param("id")
	category, err := h.DB.UpdateCategory(user.ID, id, storage.UpdateCategoryParams{
		Name:  req.Name,
		Type:  req.Type,
		Icon:  req.Icon,
		Color: req.Color,
	})
	if err != nil {
		if errors.Is(err, storage.ErrCategoryAlreadyExists) {
			log.Info("category not found", slog.String("Name", util.FromPtrOr(req.Name, "NoName")))
			writeError(
				c,
				http.StatusConflict,
				ErrCodeCategoryAlreadyExists,
				"category already exists",
			)
			return
		}

		if errors.Is(err, storage.ErrCategoryNotFound) {
			log.Info("category not found", slog.String("id", id))
			writeError(c, http.StatusNotFound, ErrCodeCategoryNotFound, "category not found")
			return
		}

		log.Error("failed to update category", logger.Error(err))
		writeError(c, http.StatusInternalServerError, ErrCodeInternal, "failed to update category")
		return
	}

	c.JSON(http.StatusOK, category)
}

func (h *Handler) DeleteCategory(c *gin.Context) {
	op := "handlers.categories.DeleteCategory"

	log := h.Logger.With(
		slog.String("op", op),
	)

	user := context.CurrentUser(c)

	id := c.Param("id")
	err := h.DB.DeleteCategory(user.ID, id)
	if err != nil {
		if errors.Is(err, storage.ErrCategoryNotFound) {
			log.Info("category not found", slog.String("id", id))
			writeError(c, http.StatusNotFound, ErrCodeCategoryNotFound, "category not found")
			return
		}

		if errors.Is(err, storage.ErrCategoryHasTransactions) {
			log.Info("category in use", slog.String("id", id))
			writeError(c, http.StatusConflict, ErrCodeCategoryInUse, "category in use")
			return
		}

		log.Error("failed to delete category", logger.Error(err))
		writeError(c, http.StatusInternalServerError, ErrCodeInternal, "failed to delete category")
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) GetCategory(c *gin.Context) {
	op := "handlers.categories.GetCategory"

	log := h.Logger.With(
		slog.String("op", op),
	)

	user := context.CurrentUser(c)

	id := c.Param("id")
	category, err := h.DB.GetCategory(user.ID, id)
	if err != nil {
		if errors.Is(err, storage.ErrCategoryNotFound) {
			log.Info("category not found", slog.String("id", id))
			writeError(c, http.StatusNotFound, ErrCodeCategoryNotFound, "category not found")
			return
		}

		log.Error("failed to get category", logger.Error(err))
		writeError(c, http.StatusInternalServerError, ErrCodeInternal, "failed to get category")
		return
	}

	c.JSON(http.StatusOK, category)
}

func (h *Handler) ListCategories(c *gin.Context) {
	op := "handlers.categories.ListCategories"

	log := h.Logger.With(
		slog.String("op", op),
	)

	user := context.CurrentUser(c)

	var params GetCategoriesQuery
	if !bindAndValidateQuery(c, log, &params) {
		return
	}

	categories, err := h.DB.GetCategories(user.ID, storage.GetCategoriesParams{
		Type: params.Type,
	})
	if err != nil {
		log.Error("failed to get categories", logger.Error(err))
		writeError(c, http.StatusInternalServerError, ErrCodeInternal, "failed to get categories")
		return
	}

	c.JSON(http.StatusOK, categories)
}
