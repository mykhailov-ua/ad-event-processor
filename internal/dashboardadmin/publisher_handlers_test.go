package dashboardadmin_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-event-processor/internal/dashboardadmin"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type publisherStub struct {
	bind dashboardadmin.PublisherBind
	err  error
}

func (s *publisherStub) ResolvePublisherBind(_ context.Context, _ uuid.UUID) (dashboardadmin.PublisherBind, error) {
	if s.err != nil {
		return dashboardadmin.PublisherBind{}, s.err
	}
	return s.bind, nil
}

func (s *publisherStub) GetPublisherDashboard(_ context.Context, _ dashboardadmin.PublisherBind, _, _ time.Time) (dashboardadmin.PublisherDashboardDTO, error) {
	return dashboardadmin.PublisherDashboardDTO{SellerID: s.bind.SellerID}, nil
}

func (s *publisherStub) ListPublisherStatements(_ context.Context, _ dashboardadmin.PublisherBind, _, _ time.Time, _, _ int32) ([]dashboardadmin.PublisherStatementDTO, int64, error) {
	return nil, 0, nil
}

func TestPublisherDashboard_requiresSellerBind(t *testing.T) {
	userID := uuid.New()
	h := &dashboardadmin.PublisherHTTPHandlers{
		Publisher: &publisherStub{err: dashboardadmin.ErrPublisherScopeRequired},
		ActorUserID: func(_ *http.Request) (uuid.UUID, bool) {
			return userID, true
		},
		WriteServiceError: func(w http.ResponseWriter, err error) {
			status, code, msg := mapPublisherTestError(err)
			httpresponse.Error(w, status, code, msg)
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publisher/dashboard", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func mapPublisherTestError(err error) (status int, code string, message string) {
	if errors.Is(err, dashboardadmin.ErrPublisherScopeRequired) {
		return http.StatusForbidden, "FORBIDDEN", err.Error()
	}
	return http.StatusInternalServerError, "INTERNAL", "internal error"
}
