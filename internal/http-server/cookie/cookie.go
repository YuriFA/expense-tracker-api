package cookie

import (
	"net/http"
	"strings"

	"github.com/yurifa/expense-tracker-api/internal/config"
)

func ParseSameSite(s string) http.SameSite {
	switch strings.ToLower(s) {
	case "strict":
		return http.SameSiteStrictMode
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteDefaultMode
	}
}

func BuildSession(cfg config.SessionConfig, value string, maxAge int) *http.Cookie {
	return &http.Cookie{ //nolint:gosec // G124: Secure is configurable to allow non-HTTPS local dev
		Name:     cfg.CookieName,
		Value:    value,
		MaxAge:   maxAge,
		Path:     "/",
		Domain:   "",
		Secure:   cfg.Secure,
		HttpOnly: true,
		SameSite: ParseSameSite(cfg.SameSite),
	}
}
