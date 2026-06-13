package requestmeta

import "context"

type requestMetaKey struct{}

var contextRequestMetaKey = requestMetaKey{}

type RequestMeta struct {
	Method     string
	Path       string
	RemoteAddr string
}

func WithRequestMeta(ctx context.Context, meta RequestMeta) context.Context {
	return context.WithValue(ctx, contextRequestMetaKey, meta)
}

func FromContext(ctx context.Context) (RequestMeta, bool) {
	meta, ok := ctx.Value(contextRequestMetaKey).(RequestMeta)
	return meta, ok
}
