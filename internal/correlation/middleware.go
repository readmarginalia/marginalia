package correlation

import (
	"net/http"
)

const headerName = "X-Correlation-ID"

func AddCorrelationId(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationId := r.Header.Get(headerName)
		if correlationId == "" {
			correlationId = createCorrelationId()
		}

		ctx := NewContext(r.Context(), correlationId)
		r = r.WithContext(ctx)

		w.Header().Set(headerName, correlationId)

		next.ServeHTTP(w, r)
	})
}
