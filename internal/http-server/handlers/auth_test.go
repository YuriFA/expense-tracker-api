package handlers_test

import (
	"net/http"
	"testing"

	"expense-tracker-api/internal/auth"
	"expense-tracker-api/internal/http-server/httperr"
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
		assert.NotEmpty(t, w.Result().Cookies())
		assert.Equal(t, "session_id", w.Result().Cookies()[0].Name)
		var response storage.User
		parseBody(t, w, &response)
		assert.Equal(t, "test@example.com", response.Email)
		assert.NotEmpty(t, response.ID)
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
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		require.Equal(t, httperr.ErrCodeUserAlreadyExists, response.Code)
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
				var response httperr.ValidationErrorResponse
				parseBody(t, w, &response)
				assert.Equal(t, httperr.ErrCodeValidationFailed, response.Code)
				assert.Equal(t, "validation failed", response.Message)
				require.Equal(t, tc.errorsLen, len(response.Errors))
				assert.Equal(t, tc.wantField, response.Errors[0].Field)
				assert.Equal(t, tc.wantMessage, response.Errors[0].Message)
			})
		}
	})
}

func TestLoginUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)
		passwordHash, err := auth.HashPassword("password123")
		require.NoError(t, err)
		_, err = db.RegisterUser(storage.RegisterUserParams{
			Email:        "test@example.com",
			PasswordHash: passwordHash,
		})
		require.NoError(t, err)
		w := performRequest(
			t,
			router,
			newJSONRequest(t, http.MethodPost, "/api/auth/login", map[string]any{
				"email":    "test@example.com",
				"password": "password123",
			}),
		)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Result().Cookies())
		assert.Equal(t, "session_id", w.Result().Cookies()[0].Name)
		var response storage.User
		parseBody(t, w, &response)
		assert.NotEmpty(t, response.ID)
		assert.Equal(t, "test@example.com", response.Email)
		assert.Empty(t, response.PasswordHash)
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := newJSONRequest(t, http.MethodPost, "/api/auth/login", map[string]any{
			"email":    "test@example.com",
			"password": "wrongpassword",
		})
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeInvalidCredentials, response.Code)
	})

	t.Run("ValidationFail", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := newJSONRequest(t, http.MethodPost, "/api/auth/login", map[string]any{
			"email": "nonemail",
		})
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response httperr.ValidationErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeValidationFailed, response.Code)
	})

	t.Run("RateLimitExceeded", func(t *testing.T) {
		router, db := setupTestEnv(t)

		passwordHash, err := auth.HashPassword("password123")
		require.NoError(t, err)
		_, err = db.RegisterUser(storage.RegisterUserParams{
			Email:        "test@example.com",
			PasswordHash: passwordHash,
		})
		require.NoError(t, err)

		for range 5 {
			req := newJSONRequest(t, http.MethodPost, "/api/auth/login", map[string]any{
				"email":    "test@example.com",
				"password": "wrongpassword",
			})
			w := performRequest(t, router, req)
			require.Equal(t, http.StatusUnauthorized, w.Code)
		}

		req := newJSONRequest(t, http.MethodPost, "/api/auth/login", map[string]any{
			"email":    "test@example.com",
			"password": "wrongpassword",
		})
		w := performRequest(t, router, req)
		require.Equal(t, http.StatusTooManyRequests, w.Code)

		req = newJSONRequest(t, http.MethodPost, "/api/auth/login", map[string]any{
			"email":    "test@example.com",
			"password": "password123",
		})
		w = performRequest(t, router, req)
		require.Equal(t, http.StatusTooManyRequests, w.Code)
	})
}

func TestLogoutUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := newAuthFixture(t)

		w := f.do(t, http.MethodPost, "/api/auth/logout", nil)
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.NotEmpty(t, w.Result().Cookies())
	})

	t.Run("NoSession", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := newJSONRequest(t, http.MethodPost, "/api/auth/logout", map[string]any{
			"email": "nonemail",
		})
		w := performRequest(t, router, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("InvalidSession", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := newJSONRequest(t, http.MethodPost, "/api/auth/logout", map[string]any{
			"email": "nonemail",
		})
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "invalid-session-id"})
		w := performRequest(t, router, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.NotEmpty(t, w.Result().Cookies())
	})

	t.Run("DoubleLogout", func(t *testing.T) {
		f := newAuthFixture(t)

		_ = f.do(t, http.MethodPost, "/api/auth/logout", nil)
		w := f.do(t, http.MethodPost, "/api/auth/logout", nil)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestMe(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := newAuthFixture(t)

		w := f.do(t, http.MethodGet, "/api/auth/me", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.User
		parseBody(t, w, &response)
		assert.NotEmpty(t, response.ID)
		assert.Equal(t, "test@example.com", response.Email)
		assert.Empty(t, response.PasswordHash)
	})

	t.Run("NoSession", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := newJSONRequest(t, http.MethodGet, "/api/auth/me", nil)
		w := performRequest(t, router, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
