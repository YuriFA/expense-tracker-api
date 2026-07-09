package handlers_test

import (
	"net/http"
	"testing"

	"expense-tracker-api/internal/http-server/handlers"
	"expense-tracker-api/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := newJSONRequest(t, http.MethodPost, "/api/auth/register", map[string]any{
			"email":    "test@example.com",
			"password": "password123",
		})
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var response storage.User
		parseBody(t, w, &response)
		assert.Equal(t, "test@example.com", response.Email)
		assert.NotEmpty(t, response.Id)
		assert.NotEmpty(t, response.CreatedAt)
		assert.NotEmpty(t, response.UpdatedAt)
	})

	t.Run("UserAlreadyExists", func(t *testing.T) {
		router, db := setupTestEnv(t)
		seedUser(t, db, "test@example.com")

		req := newJSONRequest(t, http.MethodPost, "/api/auth/register", map[string]any{
			"email":    "test@example.com",
			"password": "password123",
		})
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		require.Equal(t, handlers.ErrCodeUserAlreadyExists, response.Code)
	})

	t.Run("ValidationFail", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		cases := map[string]struct {
			body        map[string]any
			wantField   string
			wantMessage string
			errorsLen   int
		}{
			"invalid email": {
				body: map[string]any{
					"email":    "invalid-email",
					"password": "password123",
				},
				wantField:   "email",
				wantMessage: "email must be a valid email address",
				errorsLen:   1,
			},
			"short password": {
				body: map[string]any{
					"email":    "test@example.com",
					"password": "short",
				},
				wantField:   "password",
				wantMessage: "password must be at least 8 characters",
				errorsLen:   1,
			},
			"long password": {
				body: map[string]any{
					"email":    "test@example.com",
					"password": "thisisaverylongpasswordthatexceedslimitthisisaverylongpasswordthatexceedslimit",
				},
				wantField:   "password",
				wantMessage: "password must be at most 72 characters",
				errorsLen:   1,
			},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				req := newJSONRequest(t, http.MethodPost, "/api/auth/register", tc.body)
				w := performRequest(t, router, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
				var response handlers.ValidationErrorResponse
				parseBody(t, w, &response)
				assert.Equal(t, handlers.ErrCodeValidationFailed, response.Code)
				assert.Equal(t, "validation failed", response.Message)
				require.Equal(t, tc.errorsLen, len(response.Errors))
				assert.Equal(t, tc.wantField, response.Errors[0].Field)
				assert.Equal(t, tc.wantMessage, response.Errors[0].Message)
			})
		}
	})
}
