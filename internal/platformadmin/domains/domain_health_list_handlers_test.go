package domains

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubDomainHealthListService struct {
	rows []DomainHealthDTO
}

func (s *stubDomainHealthListService) ListDomainHealth(ctx context.Context) ([]DomainHealthDTO, error) {
	return s.rows, nil
}

func (s *stubDomainHealthListService) AddCustomDomain(ctx context.Context, hostname string) (DomainHealthDTO, error) {
	return DomainHealthDTO{}, nil
}

func (s *stubDomainHealthListService) DeleteCustomDomain(ctx context.Context, hostname string) error {
	return nil
}

func (s *stubDomainHealthListService) ProbeDomainNow(ctx context.Context, hostname string) (DomainHealthDTO, error) {
	return DomainHealthDTO{}, nil
}

func (s *stubDomainHealthListService) SetupDomainSSL(ctx context.Context, hostname string) (DomainSSLSetupResult, error) {
	return DomainSSLSetupResult{}, nil
}

func (s *stubDomainHealthListService) IsTLSAllowed(ctx context.Context, hostname string) (bool, error) {
	return false, nil
}

func (s *stubDomainHealthListService) ParkDomain(ctx context.Context, req ParkDomainRequest) (ParkDomainResponse, error) {
	return ParkDomainResponse{}, nil
}

func (s *stubDomainHealthListService) SetupWildcardSSL(ctx context.Context, req WildcardSSLRequest) (WildcardSSLResponse, error) {
	return WildcardSSLResponse{}, nil
}

func (s *stubDomainHealthListService) ListCloudflareZones(ctx context.Context) ([]CloudflareZone, error) {
	return nil, nil
}

func (s *stubDomainHealthListService) StartBulkParkProbe(ctx context.Context, req DomainBulkRequest) (DomainBulkJobStatus, error) {
	return DomainBulkJobStatus{}, nil
}

func (s *stubDomainHealthListService) StartBulkSSL(ctx context.Context, req DomainBulkRequest) (DomainBulkJobStatus, error) {
	return DomainBulkJobStatus{}, nil
}

func (s *stubDomainHealthListService) GetBulkJob(ctx context.Context, jobID string) (DomainBulkJobStatus, error) {
	return DomainBulkJobStatus{}, nil
}

func (s *stubDomainHealthListService) BurnDomain(ctx context.Context, hostname string, req BurnDomainRequest) (BurnDomainResponse, error) {
	return BurnDomainResponse{}, nil
}

func TestListDomains_healthFilterQuery(t *testing.T) {
	svc := &stubDomainHealthListService{
		rows: []DomainHealthDTO{
			{Hostname: "ok.example", HealthStatus: "healthy"},
			{Hostname: "burned.example", PoolStatus: "banned"},
		},
	}
	h := &DomainHealthHTTPHandlers{Service: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/domains?health_filter=burned", http.NoBody)
	w := httptest.NewRecorder()
	h.listDomains(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var got []DomainHealthDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Len(t, got, 1)
	require.Equal(t, "burned.example", got[0].Hostname)
}
