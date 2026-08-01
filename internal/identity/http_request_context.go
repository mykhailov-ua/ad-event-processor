package identity

import (
	"context"
	"net/http"
)

type httpRequestContextKey struct{}

// WithHTTPRequest attaches the originating HTTP request for in-process auth calls.
func WithHTTPRequest(ctx context.Context, r *http.Request) context.Context {
	if r == nil {
		return ctx
	}
	return context.WithValue(ctx, httpRequestContextKey{}, r)
}

func httpRequestFromContext(ctx context.Context) (*http.Request, bool) {
	r, ok := ctx.Value(httpRequestContextKey{}).(*http.Request)
	return r, ok && r != nil
}
