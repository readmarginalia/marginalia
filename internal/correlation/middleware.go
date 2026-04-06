package correlation

import (
	"marginalia/internal/telemetry/logging"
	"net/http"
)

const headerName = "X-Correlation-ID"

func AddCorrelationId(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationId := r.Header.Get(headerName)
		if correlationId == "" {
			correlationId = createCorrelationId()
		}

		ctx := WithCorrelationId(r.Context(), correlationId)
		logger := logging.WithCorrelationId(ctx, correlationId)
		ctx = logging.WithLogger(ctx, logger)
		r = r.WithContext(ctx)

		w.Header().Set(headerName, correlationId)

		next.ServeHTTP(w, r)
	})
}
