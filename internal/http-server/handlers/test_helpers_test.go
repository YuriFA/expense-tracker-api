package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"expense-tracker-api/internal/auth"
	"expense-tracker-api/internal/config"
	httpserver "expense-tracker-api/internal/http-server"
	"expense-tracker-api/internal/http-server/handlers"
	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func setupTestEnv(t *testing.T) (*gin.Engine, *sqlite.Storage) {
	t.Helper()
	db := sqlite.NewTestDB(t)
	// NOTE: For debugging, you can use a real logger instead of the discard logger.
	// log := logger.New(logger.Options{Environment: "dev"})
	log := logger.NewDiscardLogger()
	cfg := &config.HTTPServer{
		Address:      ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  30 * time.Second,
		CorsConfig: config.CORSConfig{
			AllowedOrigins: []string{"*"},
		},
		SessionConfig: config.SessionConfig{
			TTL:        24 * time.Hour,
			CookieName: "session_id",
			Secure:     false,
			SameSite:   "lax",
		},
	}
	h := handlers.NewHandler(log, db, cfg, auth.NewLoginRateLimiter(5, time.Minute))
	return httpserver.NewRouter(log, db, h, cfg), db
}

func newJSONRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	paramsJson, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, bytes.NewReader(paramsJson))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func performRequest(
	t *testing.T,
	router *gin.Engine,
	req *http.Request,
) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

func parseBody[T any](t *testing.T, recorder *httptest.ResponseRecorder, target *T) {
	t.Helper()
	err := json.Unmarshal(recorder.Body.Bytes(), target)
	require.NoError(t, err)
}

type authFixture struct {
	Router *gin.Engine
	DB     *sqlite.Storage
	User   *storage.User
	Cookie *http.Cookie
}

func newAuthFixture(t *testing.T) *authFixture {
	t.Helper()
	router, db := setupTestEnv(t)
	user := seedUser(t, db, "test@example.com")
	cookie := createSessionCookie(t, db, user.ID)
	return &authFixture{Router: router, DB: db, User: user, Cookie: cookie}
}

func createSessionCookie(t *testing.T, db *sqlite.Storage, userID string) *http.Cookie {
	t.Helper()
	sessionID, err := auth.GenerateSessionToken()
	require.NoError(t, err)
	_, err = db.CreateSession(storage.CreateSessionParams{
		SessionID: sessionID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)
	return &http.Cookie{Name: "session_id", Value: sessionID}
}

// do — выполняет HTTP запрос с auth cookie
func (f *authFixture) do(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = newJSONRequest(t, method, path, body)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.AddCookie(f.Cookie)
	return performRequest(t, f.Router, req)
}

func seedUser(t *testing.T, db *sqlite.Storage, email string) *storage.User {
	t.Helper()
	user, err := db.RegisterUser(storage.RegisterUserParams{
		Email:        email,
		PasswordHash: "strongpasswordhash",
	})
	require.NoError(t, err)
	return user
}

func defaultAccountParams(userID string) storage.CreateAccountParams {
	return storage.CreateAccountParams{
		UserID:         userID,
		Name:           "Bank",
		Currency:       "USD",
		OpeningBalance: 10000,
	}
}

func seedAccount(
	t *testing.T,
	db *sqlite.Storage,
	params storage.CreateAccountParams,
) *storage.Account {
	t.Helper()
	account, err := db.CreateAccount(params)
	require.NoError(t, err)
	return account
}

func defaultCategoryParams(userID string) storage.CreateCategoryParams {
	return storage.CreateCategoryParams{
		UserID: userID,
		Name:   "DefaultIncomeCategory",
		Type:   "income",
		Icon:   "🍔",
		Color:  "#FF0000",
	}
}

func seedCategory(
	t *testing.T,
	db *sqlite.Storage,
	params storage.CreateCategoryParams,
) *storage.Category {
	category, err := db.CreateCategory(params)
	require.NoError(t, err)
	return category
}

func seedDefaultIncomeCategory(
	t *testing.T,
	db *sqlite.Storage,
	userID string,
) *storage.Category {
	params := defaultCategoryParams(userID)
	params.Type = "income"
	params.Name = uuid.NewString()[:8] + "DefaultIncomeCategory"
	category, err := db.CreateCategory(params)
	require.NoError(t, err)
	return category
}

func seedDefaultExpenseCategory(
	t *testing.T,
	db *sqlite.Storage,
	userID string,
) *storage.Category {
	params := defaultCategoryParams(userID)
	params.Type = "expense"
	params.Name = uuid.NewString()[:8] + "DefaultExpenseCategory"
	category, err := db.CreateCategory(params)
	require.NoError(t, err)
	return category
}

func defaultCashflowTransactionParams(
	userID, accountID, categoryID string,
) storage.CreateTransactionParams {
	return storage.CreateTransactionParams{
		UserID:     userID,
		Type:       "expense",
		Amount:     1000,
		AccountID:  &accountID,
		CategoryID: &categoryID,
		OccurredAt: time.Now(),
	}
}

func defaultTransferTransactionParams(
	userID, fromAccountID, toAccountID string,
) storage.CreateTransactionParams {
	return storage.CreateTransactionParams{
		UserID:        userID,
		Type:          "transfer",
		Amount:        1000,
		FromAccountID: &fromAccountID,
		ToAccountID:   &toAccountID,
		OccurredAt:    time.Now(),
	}
}

func seedTransaction(
	t *testing.T,
	db *sqlite.Storage,
	params storage.CreateTransactionParams,
) *storage.Transaction {
	t.Helper()
	transaction, err := db.CreateTransaction(params)
	require.NoError(t, err)
	return transaction
}

type seedCommonTransactionParams struct {
	userID          string
	categoryName    string
	transactionType string
}

func seedCommonTransaction(
	t *testing.T,
	db *sqlite.Storage,
	params seedCommonTransactionParams,
) *storage.Transaction {
	t.Helper()

	var transaction *storage.Transaction
	switch params.transactionType {
	case "income", "expense":
		categoryParams := defaultCategoryParams(params.userID)
		categoryParams.Name = params.categoryName
		categoryParams.Type = params.transactionType
		category := seedCategory(t, db, categoryParams)
		accountParams := defaultAccountParams(params.userID)
		accountParams.Name = "Cash"
		account := seedAccount(t, db, accountParams)
		transactionParams := defaultCashflowTransactionParams(
			params.userID,
			account.ID,
			category.ID,
		)
		transactionParams.Type = params.transactionType
		transactionParams.Amount = 100000
		transaction = seedTransaction(t, db, transactionParams)
	case "transfer":
		accountFromParams := defaultAccountParams(params.userID)
		accountFromParams.Name = "Bank"
		accountFromParams.OpeningBalance = 50000
		accountFrom := seedAccount(t, db, accountFromParams)
		accountToParams := defaultAccountParams(params.userID)
		accountToParams.Name = "Cash"
		accountToParams.OpeningBalance = 20000
		accountTo := seedAccount(t, db, accountToParams)
		transactionParams := defaultTransferTransactionParams(
			params.userID,
			accountFrom.ID,
			accountTo.ID,
		)
		transactionParams.Type = params.transactionType
		transactionParams.Amount = 30000
		transaction = seedTransaction(t, db, transactionParams)
	default:
		t.Fatalf("unsupported transaction type: %s", params.transactionType)
		return nil
	}

	return transaction
}

type seedTransactionAtParams struct {
	userID          string
	transactionType string
	categoryName    string
	occurredAt      time.Time
	amount          int64
}

func seedTransactionAt(
	t *testing.T,
	db *sqlite.Storage,
	params seedTransactionAtParams,
) *storage.Transaction {
	t.Helper()

	var transaction *storage.Transaction
	switch params.transactionType {
	case "income", "expense":
		categoryParams := defaultCategoryParams(params.userID)
		categoryParams.Name = params.categoryName
		categoryParams.Type = params.transactionType
		category := seedCategory(t, db, categoryParams)
		accountParams := defaultAccountParams(params.userID)
		accountParams.Name = "Cash"
		account := seedAccount(t, db, accountParams)
		transaction = seedTransaction(t, db, storage.CreateTransactionParams{
			UserID:      params.userID,
			Type:        params.transactionType,
			Amount:      params.amount,
			Description: "Common transaction",
			OccurredAt:  params.occurredAt,
			AccountID:   &account.ID,
			CategoryID:  &category.ID,
		})
	case "transfer":
		fromAccountParams := defaultAccountParams(params.userID)
		fromAccountParams.Name = "Bank"
		fromAccountParams.OpeningBalance = 50000
		fromAccount := seedAccount(t, db, fromAccountParams)
		toAccountParams := defaultAccountParams(params.userID)
		toAccountParams.Name = "Cash"
		toAccountParams.OpeningBalance = 20000
		toAccount := seedAccount(t, db, toAccountParams)
		transaction = seedTransaction(t, db, storage.CreateTransactionParams{
			UserID:        params.userID,
			Type:          params.transactionType,
			Amount:        params.amount,
			Description:   "Common transfer",
			OccurredAt:    params.occurredAt,
			FromAccountID: &fromAccount.ID,
			ToAccountID:   &toAccount.ID,
		})
	default:
		t.Fatalf("unsupported transaction type: %s", params.transactionType)
		return nil
	}

	return transaction
}
