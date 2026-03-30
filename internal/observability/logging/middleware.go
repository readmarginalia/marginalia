package logging

import (
	"log/slog"
	"net/http"

	"time"
)

func AddRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		attrs := []any{
			"http.method", r.Method,
			"http.path", r.URL.Path,
			"http.remote_addr", r.RemoteAddr,
		}

		logger := slog.Default().With(attrs...)
		ctx := WithLogger(r.Context(), logger)
		r = r.WithContext(ctx)
		logger.InfoContext(ctx, "request started")
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		logger.InfoContext(ctx, "request completed", "duration_ms", duration.Milliseconds())
	})
}
