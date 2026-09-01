package platformadmin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/reports"

	"github.com/stretchr/testify/require"
)

func TestGetSessionBootstrap_returnsUserSessionAndEula(t *testing.T) {
	h := &SessionHTTPHandlers{
		ResolveAuthUser: func(ctx context.Context) (BootstrapUserDTO, bool) {
			return BootstrapUserDTO{
				ID:          "11111111-1111-1111-1111-111111111111",
				Role:        "A",
				CustomerID:  "22222222-2222-2222-2222-222222222222",
				Permissions: []string{"settings:write"},
			}, true
		},
		ResolveUser: func(ctx context.Context) (SessionUser, bool) {
			return SessionUser{Role: "A"}, true
		},
		EulaSnapshot: func(ctx context.Context) (EulaBootstrapDTO, error) {
			return EulaBootstrapDTO{
				EulaRequired: true,
				EulaAccepted: false,
				EulaVersion:  "2026-01",
			}, nil
		},
		Freshness: func(ctx context.Context) reports.DataFreshnessDTO {
			return reports.DataFreshnessDTO{}
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/session/bootstrap", http.NoBody)
	rec := httptest.NewRecorder()
	h.getSessionBootstrap(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var body SessionBootstrapDTO
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.Equal(t, "A", body.User.Role)
	require.Equal(t, "A", body.Session.Role)
	require.True(t, body.EulaRequired)
	require.False(t, body.EulaAccepted)
}

func TestGetSessionBootstrap_holdoutRequiresAuthenticatedUser(t *testing.T) {
	h := &SessionHTTPHandlers{
		ResolveAuthUser: func(ctx context.Context) (BootstrapUserDTO, bool) {
			return BootstrapUserDTO{}, false
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/session/bootstrap", http.NoBody)
	rec := httptest.NewRecorder()
	h.getSessionBootstrap(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
