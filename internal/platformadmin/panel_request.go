package platformadmin

import (
	"context"
	"net/http"
)

type panelRequestKey struct{}

func WithPanelRequest(ctx context.Context, r *http.Request) context.Context {
	if r == nil {
		return ctx
	}
	return context.WithValue(ctx, panelRequestKey{}, r)
}

func PanelRequestFromContext(ctx context.Context) *http.Request {
	if r, ok := ctx.Value(panelRequestKey{}).(*http.Request); ok {
		return r
	}
	return nil
}
