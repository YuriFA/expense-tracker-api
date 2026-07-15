package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"expense-tracker-api/internal/http-server/keys"
	"expense-tracker-api/internal/http-server/middleware"
	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedUser(t *testing.T, db *sqlite.Storage, email string) *storage.User {
	t.Helper()
	user, err := db.RegisterUser(storage.RegisterUserParams{
		Email:        email,
		PasswordHash: "strongpasswordhash",
	})
	require.NoError(t, err)
	return user
}

type idemFixture struct {
	router    *gin.Engine
	db        *sqlite.Storage
	user      *storage.User
	handlerMu sync.Mutex
	callCount int
}

func newIdemFixture(t *testing.T) *idemFixture {
	t.Helper()
	db := sqlite.NewTestDB(t)
	user := seedUser(t, db, "test@example.com")
	log := logger.NewDiscardLogger()

	f := &idemFixture{
		db:   db,
		user: user,
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(keys.CurrentUserKey, f.user)
		c.Next()
	})
	r.POST(
		"/transactions",
		middleware.Idempotency(db, log),
		func(c *gin.Context) {
			f.handlerMu.Lock()
			f.callCount++
			f.handlerMu.Unlock()
			c.JSON(http.StatusCreated, gin.H{"id": "tx-1"})
		},
	)
	f.router = r
	return f
}

func (f *idemFixture) do(t *testing.T, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

func TestIdempotency(t *testing.T) {
	t.Run("missing header returns 400", func(t *testing.T) {
		f := newIdemFixture(t)
		w := f.do(t, "", `{"x":1}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 0, f.callCount)
	})

	t.Run("first request is executed", func(t *testing.T) {
		f := newIdemFixture(t)
		w := f.do(t, "k1", `{"x":1}`)
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, 1, f.callCount)
	})

	t.Run("replay with same key and body returns cached response", func(t *testing.T) {
		f := newIdemFixture(t)
		body := `{"x":1}`
		w1 := f.do(t, "k1", body)
		w2 := f.do(t, "k1", body)
		assert.Equal(t, http.StatusCreated, w1.Code)
		assert.Equal(t, http.StatusCreated, w2.Code)
		assert.Equal(t, w1.Body.String(), w2.Body.String())
		assert.Equal(t, 1, f.callCount) // handler вызвался ОДИН раз
	})

	t.Run("same key different body returns 409", func(t *testing.T) {
		f := newIdemFixture(t)
		_ = f.do(t, "k1", `{"x":1}`)
		w := f.do(t, "k1", `{"x":2}`)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Equal(t, 1, f.callCount)
	})

	t.Run("different users can reuse same key", func(t *testing.T) {
		f1 := newIdemFixture(t)
		f2 := newIdemFixture(t)
		_ = f1.do(t, "shared-key", `{"x":1}`)
		w := f2.do(t, "shared-key", `{"x":1}`)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestIdempotency_Concurrent(t *testing.T) {
	f := newIdemFixture(t)
	body := `{"x":1}`
	const N = 16
	var wg sync.WaitGroup
	wg.Add(N)
	responses := make([]*httptest.ResponseRecorder, N)
	start := make(chan struct{})
	for i := range N {
		go func(i int) {
			defer wg.Done()
			<-start // выравниваем старт
			responses[i] = f.do(t, "concurrent-key", body)
		}(i)
	}
	close(start)
	wg.Wait()

	require.Equal(t, 1, f.callCount,
		"handler must run exactly once under concurrent identical requests")

	ok := 0
	conflict := 0
	var firstOKBody string
	for _, w := range responses {
		switch w.Code {
		case http.StatusCreated:
			ok++
			if firstOKBody == "" {
				firstOKBody = w.Body.String()
			} else {
				assert.Equal(t, firstOKBody, w.Body.String(),
					"all successful responses must be byte-identical")
			}
		case http.StatusConflict:
			conflict++ // pending-ветка честно отдаёт 409
		default:
			t.Errorf("unexpected status %d", w.Code)
		}
	}
	require.GreaterOrEqual(t, ok, 1, "at least one request must return the handler response")
	require.Equal(t, N-ok, conflict, "remaining requests must hit the pending-409 path")
}
