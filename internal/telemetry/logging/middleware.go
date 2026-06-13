package logging

import (
	"fmt"
	"log/slog"
	"net/http"

	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

func AddRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

		logger := slog.Default()
		ctx := r.Context()
		logger.InfoContext(ctx, fmt.Sprintf("request starting: %s: %s ", r.Method, r.URL.Path))

		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		status := ww.Status()
		size := ww.BytesWritten()

		logger.InfoContext(ctx, fmt.Sprintf("request completed in %vms, Status: %d", duration.Milliseconds(), status),
			"status", status,
			"request.duration", duration.Milliseconds(),
			"response.size.bytes", size,
		)
	})
}
