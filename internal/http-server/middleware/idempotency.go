package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"expense-tracker-api/internal/http-server/context"
	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"
	"expense-tracker-api/internal/util"

	"github.com/gin-gonic/gin"
)

type bodyRecorder struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// pendingStaleThreshold is how long a pending row may exist before we treat it
// as orphaned (the original request likely crashed between CreateIdempotencyKey
// and UpdateIdempotencyKey). Past this age, the row is deleted and the caller
// is allowed to claim the key afresh.
const pendingStaleThreshold = 5 * time.Minute

// idempotencyKeyTTL is how long a recorded key (and its cached response) is
// kept after creation. Past this age the row is eligible for cleanup by
// DeleteExpiredIdempotencyKeys and is treated as expired on read.
const idempotencyKeyTTL = 24 * time.Hour

func (r *bodyRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// isPendingStale reports whether the given createdAt timestamp is older than
// pendingStaleThreshold. On parse failure returns false (treat as live).
func isPendingStale(createdAt string) bool {
	t, err := util.ParseDatetime(createdAt)
	if err != nil {
		return false
	}
	return time.Since(t) > pendingStaleThreshold
}

func Idempotency(db *sqlite.Storage, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := log.With(slog.String("op", "httpserver.middleware.Idempotency"))

		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			log.Info("missing idempotency key")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing idempotency key"})
			return
		}

		user := context.CurrentUser(c)

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				gin.H{"code": "INVALID_REQUEST", "error": "failed to read request body"},
			)
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		hash := sha256.Sum256(bodyBytes)
		hashStr := hex.EncodeToString(hash[:])

		ik, err := db.CreateIdempotencyKey(storage.CreateIdempotencyKeyParams{
			IdempotencyKey: key,
			UserID:         user.ID,
			RequestHash:    hashStr,
			ExpiresAt:      time.Now().UTC().Add(idempotencyKeyTTL),
		})
		if err != nil {
			if errors.Is(err, storage.ErrIdempotencyKeyInUse) {
				if existing, _ := db.GetByUserAndKey(user.ID, key); existing != nil {
					if dispatchExisting(c, db, log, existing, user.ID, hashStr) {
						return
					}
				}
				c.AbortWithStatusJSON(
					http.StatusConflict,
					gin.H{"error": "idempotency key already used"},
				)
				return
			}
			log.Info("failed to create idempotency key", logger.Error(err))
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": "internal server error"},
			)
			return
		}

		rec := &bodyRecorder{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = rec
		c.Next()

		persistResponse(db, log, ik.ID, user.ID, rec)
	}
}

func dispatchExisting(
	c *gin.Context,
	db *sqlite.Storage,
	l *slog.Logger,
	ik *storage.IdempotencyKey,
	userID string,
	hashStr string,
) bool {
	expiresAt, err := util.ParseDatetime(ik.ExpiresAt)
	if err != nil {
		l.Error("failed to parse idempotency key expiration date",
			slog.String("error", err.Error()))
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{"error": "internal server error"},
		)
		return true
	}

	if expiresAt.Before(time.Now().UTC()) {
		_ = db.DeleteIdempotencyKey(userID, ik.ID)
		return false
	}

	switch ik.Status {
	case "pending":
		if isPendingStale(ik.CreatedAt) {
			_ = db.DeleteIdempotencyKey(userID, ik.ID)
			return false
		}
		c.AbortWithStatusJSON(
			http.StatusConflict,
			gin.H{"error": "idempotency key already used"},
		)
		return true
	case "completed":
		if hashStr != ik.RequestHash {
			c.AbortWithStatusJSON(
				http.StatusConflict,
				gin.H{"error": "idempotency key request hash mismatch"},
			)
			return true
		}
		replayResponse(c, ik)
		c.Abort()
		return true
	case "failed":
		_ = db.DeleteIdempotencyKey(userID, ik.ID)
		return false
	}
	return false
}

func replayResponse(c *gin.Context, ik *storage.IdempotencyKey) {
	if ik.ResponseHeaders != nil {
		var headers http.Header
		if err := json.Unmarshal([]byte(*ik.ResponseHeaders), &headers); err == nil {
			for k, vs := range headers {
				for _, v := range vs {
					c.Writer.Header().Add(k, v)
				}
			}
		}
	}
	status := http.StatusOK
	if ik.ResponseStatus != nil {
		status = *ik.ResponseStatus
	}
	c.Data(status, "", ik.ResponseBody)
}

func persistResponse(
	db *sqlite.Storage,
	l *slog.Logger,
	ikID string,
	userID string,
	rec *bodyRecorder,
) {
	resStatus := rec.Status()
	resBody := rec.body.Bytes()
	filtered := filterResponseHeaders(rec.Header())
	jsonHeaders, err := json.Marshal(filtered)
	if err != nil {
		l.Info("failed to marshal response headers", logger.Error(err))
		return
	}

	status := "completed"
	if resStatus >= 400 {
		status = "failed"
	}
	if _, err := db.UpdateIdempotencyKey(userID, ikID, storage.UpdateIdempotencyKeyParams{
		Status:          &status,
		ResponseStatus:  &resStatus,
		ResponseHeaders: &jsonHeaders,
		ResponseBody:    &resBody,
	}); err != nil {
		l.Info("failed to update idempotency key", logger.Error(err))
	}
}

func filterResponseHeaders(h http.Header) http.Header {
	out := h.Clone()
	for k := range out {
		switch k {
		case "Content-Type", "Content-Length", "Location":
		default:
			out.Del(k)
		}
	}
	return out
}
