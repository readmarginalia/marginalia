package logging

import (
	"fmt"
	"net/http"

	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

func AddRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		attrs := []any{
			"http.request.method", r.Method,
			"url.path", r.URL.Path,
			"client.address", r.RemoteAddr,
		}

		logger := FromContext(r.Context()).With(attrs...)
		ctx := WithLogger(r.Context(), logger)
		r = r.WithContext(ctx)
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
