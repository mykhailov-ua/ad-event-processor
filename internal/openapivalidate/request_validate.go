package openapivalidate

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"ad-event-processor/pkg/httpresponse"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
)

const (
	openAPIDir = "api/openapi"
	bundleRel  = "openapi.bundle.yaml"
)

type RequestValidationOptions struct {
	Enabled    bool
	BundlePath string
}

func NewRequestValidationMiddleware(ctx context.Context, opts RequestValidationOptions) (func(http.Handler) http.Handler, error) {
	if !opts.Enabled {
		return passthroughMiddleware(), nil
	}
	if strings.TrimSpace(opts.BundlePath) == "" {
		return nil, fmt.Errorf("openapi request validation enabled but bundle path is empty")
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false
	doc, err := loader.LoadFromFile(opts.BundlePath)
	if err != nil {
		return nil, fmt.Errorf("openapi request validation: load bundle: %w", err)
	}
	if err := doc.Validate(ctx); err != nil {
		return nil, fmt.Errorf("openapi request validation: validate bundle: %w", err)
	}

	router, err := legacy.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("openapi request validation: router: %w", err)
	}

	filterOpts := &openapi3filter.Options{
		ExcludeRequestBody:  false,
		ExcludeResponseBody: true,
		MultiError:          false,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route, pathParams, err := router.FindRoute(r)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if !requestValidationEnabled(route) {
				next.ServeHTTP(w, r)
				return
			}
			if r.Body == nil {
				r.Body = http.NoBody
			}
			input := &openapi3filter.RequestValidationInput{
				Request:    r,
				PathParams: pathParams,
				Route:      route,
				Options:    filterOpts,
			}
			if err := openapi3filter.ValidateRequest(r.Context(), input); err != nil {
				httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", validationErrorMessage(err))
				return
			}
			next.ServeHTTP(w, r)
		})
	}, nil
}

func passthroughMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return next }
}

func requestValidationEnabled(route *routers.Route) bool {
	if route == nil || route.Operation == nil {
		return false
	}
	_, ok := RequestValidationOperationIDs[route.Operation.OperationID]
	return ok
}

func validationErrorMessage(err error) string {
	if err == nil {
		return "request validation failed"
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "request validation failed"
	}
	if len(msg) > 240 {
		return msg[:240]
	}
	return msg
}

func DefaultBundlePath(repoRoot string) string {
	return filepath.Join(repoRoot, openAPIDir, bundleRel)
}

func ResolveRequestValidationMiddleware(ctx context.Context, repoRoot string, enabled bool) (func(http.Handler) http.Handler, error) {
	mw, err := NewRequestValidationMiddleware(ctx, RequestValidationOptions{
		Enabled:    enabled,
		BundlePath: DefaultBundlePath(repoRoot),
	})
	if err != nil {
		if enabled {
			return nil, err
		}
		slog.Warn("openapi request validation disabled after bundle load failure", "error", err)
		return passthroughMiddleware(), nil
	}
	if enabled {
		slog.Info("openapi request validation enabled", "bundle", DefaultBundlePath(repoRoot), "operations", len(RequestValidationOperationIDs))
	}
	return mw, nil
}
