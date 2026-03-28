package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"log"
	"net/http"
	"strings"
	"time"
)

// tokenAuth returns middleware that validates Bearer token authentication
// and optionally rate-limits failed attempts.
func tokenAuth(cfg AuthConfig, limiter *failedAuthLimiter) func(http.Handler) http.Handler {
	expectedHash := sha256.Sum256([]byte(cfg.Token))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientID, proxied := cfg.clientIdentity(r)

			if limiter != nil {
				if blockedUntil, blocked := limiter.Blocked(clientID, time.Now()); blocked {
					logAuthDenied(r, clientID, proxied, "rate_limited", blockedUntil)
					jsonError(w, "too many failed authentication attempts", http.StatusTooManyRequests)
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
					jsonError(w, "too many failed authentication attempts", http.StatusLocked)
					return
				}
			}

			logAuthDenied(r, clientID, proxied, "invalid_token", time.Time{})
			jsonError(w, "unauthorized", http.StatusUnauthorized)
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

func logAuthDenied(r *http.Request, clientID string, proxied bool, reason string, blockedUntil time.Time) {
	if blockedUntil.IsZero() {
		log.Printf("auth denied: method=%s path=%s client=%s proxied=%t reason=%s", r.Method, r.URL.Path, clientID, proxied, reason)
		return
	}
	log.Printf("auth denied: method=%s path=%s client=%s proxied=%t reason=%s blocked_until=%s", r.Method, r.URL.Path, clientID, proxied, reason, blockedUntil.UTC().Format(time.RFC3339))
}
