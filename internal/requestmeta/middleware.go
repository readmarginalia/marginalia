package requestmeta

import (
	"marginalia/internal/configuration"
	mg_http "marginalia/internal/infra/http"
	"net/http"
)

func AddRequestMetadata(cfg configuration.AppConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			remoteAddress, _ := mg_http.ClientIdentity(r, cfg)
			meta := RequestMeta{
				Method:     r.Method,
				Path:       r.URL.Path,
				RemoteAddr: remoteAddress,
			}

			ctx := WithRequestMeta(r.Context(), meta)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
