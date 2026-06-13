package requestmeta

import (
	"net/http"
)

func AddRequestMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta := RequestMeta{
			Method:     r.Method,
			Path:       r.URL.Path,
			RemoteAddr: r.RemoteAddr,
		}

		ctx := WithRequestMeta(r.Context(), meta)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
