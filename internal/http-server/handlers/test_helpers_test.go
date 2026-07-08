package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"expense-tracker-api/internal/config"
	httpserver "expense-tracker-api/internal/http-server"
	"expense-tracker-api/internal/http-server/handlers"
	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupTestEnv(t *testing.T) (*gin.Engine, *sqlite.Storage) {
	t.Helper()
	db := sqlite.NewTestDB(t)
	log := logger.NewDiscardLogger()
	h := handlers.NewHandler(log, db)
	return httpserver.NewRouter(log, h, config.HTTPServer{
		CorsConfig: config.CORSConfig{
			AllowedOrigins: []string{"*"},
		},
	}), db
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

func seedAccount(
	t *testing.T,
	db *sqlite.Storage,
	name string,
	openingBalance int64,
) *storage.Account {
	account, err := db.CreateAccount(storage.CreateAccountParams{
		Name:           name,
		Currency:       "USD",
		OpeningBalance: openingBalance,
	})
	require.NoError(t, err)
	return account
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

func seedCommonCategoryAndAccount(
	t *testing.T,
	db *sqlite.Storage,
	categoryType string,
) (*storage.Category, *storage.Account) {
	t.Helper()

	category := seedCategory(t, db, storage.CreateCategoryParams{
		Name:  "Salary",
		Type:  categoryType,
		Icon:  "dollar-sign",
		Color: "green",
	})
	account := seedAccount(t, db, "Cash", 100000)

	return category, account
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

func seedCommonTransaction(
	t *testing.T,
	db *sqlite.Storage,
	transactionType string,
) *storage.Transaction {
	t.Helper()

	occurredAt := time.Now()

	var transaction *storage.Transaction
	switch transactionType {
	case "income", "expense":
		category, account := seedCommonCategoryAndAccount(t, db, transactionType)

		transaction = seedTransaction(t, db, storage.CreateTransactionParams{
			Type:        transactionType,
			Amount:      100000,
			Description: "Common transaction",
			OccurredAt:  occurredAt,
			AccountId:   &account.Id,
			CategoryId:  &category.Id,
		})
	case "transfer":
		accountFrom := seedAccount(t, db, "Bank", 50000)
		accountTo := seedAccount(t, db, "Cash", 20000)
		transaction = seedTransaction(t, db, storage.CreateTransactionParams{
			Type:          transactionType,
			Amount:        30000,
			Description:   "Common transfer",
			OccurredAt:    occurredAt,
			FromAccountId: &accountFrom.Id,
			ToAccountId:   &accountTo.Id,
		})
	default:
		t.Fatalf("unsupported transaction type: %s", transactionType)
		return nil
	}

	return transaction
}

func seedTransactionAt(
	t *testing.T,
	db *sqlite.Storage,
	transactionType string,
	occurredAt time.Time,
	amount int64,
) *storage.Transaction {
	t.Helper()

	var transaction *storage.Transaction
	switch transactionType {
	case "income", "expense":
		category, account := seedCommonCategoryAndAccount(t, db, transactionType)
		transaction = seedTransaction(t, db, storage.CreateTransactionParams{
			Type:        transactionType,
			Amount:      amount,
			Description: "Common transaction",
			OccurredAt:  occurredAt,
			AccountId:   &account.Id,
			CategoryId:  &category.Id,
		})
	case "transfer":
		fromAccount := seedAccount(t, db, "Bank", 50000)
		toaccount := seedAccount(t, db, "Cash", 20000)
		transaction = seedTransaction(t, db, storage.CreateTransactionParams{
			Type:          transactionType,
			Amount:        amount,
			Description:   "Common transfer",
			OccurredAt:    occurredAt,
			FromAccountId: &fromAccount.Id,
			ToAccountId:   &toaccount.Id,
		})
	default:
		t.Fatalf("unsupported transaction type: %s", transactionType)
		return nil
	}

	return transaction
}
