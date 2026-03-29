package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"log/slog"
	"marginalia/internal/infra/http"
	stdhttp "net/http"
	"strings"
	"time"
)

// TokenAuth returns middleware that validates Bearer token authentication
// and optionally rate-limits failed attempts.
func TokenAuth(cfg AuthConfig, limiter *http.FailedAuthLimiter) func(stdhttp.Handler) stdhttp.Handler {
	expectedHash := sha256.Sum256([]byte(cfg.Token))
	return func(next stdhttp.Handler) stdhttp.Handler {
		return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			clientID, proxied := cfg.clientIdentity(r)

			if limiter != nil {
				if blockedUntil, blocked := limiter.Blocked(clientID, time.Now()); blocked {
					logAuthDenied(r, clientID, proxied, "rate_limited", blockedUntil)
					http.JsonError(w, "too many failed authentication attempts", stdhttp.StatusTooManyRequests)
					return
				}
			}

			providedToken, ok := bearerToken(r.Header.Get("Authorization"))
			if ok && constantTimeMatch(providedToken, expectedHash) {
				next.ServeHTTP(w, r)
				return
			}

			if limiter != nil {
				if blockedUntil, newlyBlocked := limiter.CheckAndRecord(clientID, time.Now()); newlyBlocked {
					logAuthDenied(r, clientID, proxied, "rate_limited", blockedUntil)
					http.JsonError(w, "too many failed authentication attempts", stdhttp.StatusTooManyRequests)
					return
				}
			}

			logAuthDenied(r, clientID, proxied, "invalid_token", time.Time{})
			http.JsonError(w, "unauthorized", stdhttp.StatusUnauthorized)
		})
	}
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	return parts[1], true
}

func constantTimeMatch(provided string, expectedHash [32]byte) bool {
	if provided == "" {
		return false
	}
	h := sha256.Sum256([]byte(provided))
	return subtle.ConstantTimeCompare(h[:], expectedHash[:]) == 1
}

func logAuthDenied(r *stdhttp.Request, clientID string, proxied bool, reason string, blockedUntil time.Time) {
	if blockedUntil.IsZero() {
		slog.Error("auth denied",
			"method", r.Method,
			"path", r.URL.Path,
			"client", clientID,
			"proxied", proxied,
			"reason", reason)
		return
	}
	slog.Error("auth denied",
		"method", r.Method,
		"path", r.URL.Path,
		"client", clientID,
		"proxied", proxied,
		"reason", reason,
		"blocked_until", blockedUntil.UTC().Format(time.RFC3339))
}
