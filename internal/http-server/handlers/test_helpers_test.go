package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httpserver "expense-tracker-api/internal/http-server"
	"expense-tracker-api/internal/http-server/handlers"
	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage/sqlite"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// setupTestEnv(t) *gin.Engine, *sqlite.Storage 	Собирает router с in-memory SQLite + discard logger. Единая точка setup'а.
// newJSONRequest(t, method, path, body) *http.Request 	Конструирует request с JSON-телом, выставляет Content-Type.
// performRequest(t, router, req) *httptest.ResponseRecorder 	Выполняет request, возвращает recorder.
// parseBody(t, recorder, target *T) 	Декодирует JSON-ответ в типизированный target.
// assertErrorResponse(t, code, body, errCode, errMsg) 	Проверяет формат writeError (тот самый error-response контракт).
// assertValidationError(t, code, body, expectedFields ...) 	Проверяет формат writeValidationError.

func setupTestEnv(t *testing.T) (*gin.Engine, *sqlite.Storage) {
	t.Helper()
	db := sqlite.NewTestDB(t)
	log := logger.NewDiscardLogger()
	h := handlers.NewHandler(log, db)
	return httpserver.NewRouter(h), db
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
