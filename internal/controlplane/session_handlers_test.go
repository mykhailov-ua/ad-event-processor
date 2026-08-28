package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/controlplane/authz"

	"github.com/stretchr/testify/require"
)

func TestBuildSessionNav_buyerOmitsOpsLink(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Mask: authz.MaskMasked,
	})
	items := buildSessionNav(ctx)
	for _, item := range items {
		require.NotEqual(t, "/ops", item.Href)
		require.NotEqual(t, "/customers", item.Href)
	}
}

func TestSessionRoute_returnsNavItems(t *testing.T) {
	t.Parallel()
	h := &SessionHTTPHandlers{}
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/session", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body SessionResponseDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotEmpty(t, body.NavItems)
}
