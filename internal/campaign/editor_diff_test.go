package campaign

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type diffCampaignStub struct {
	patchRevisionCampaignStub
	byID map[uuid.UUID]CampaignDTO
}

func (s *diffCampaignStub) GetCampaign(_ context.Context, id uuid.UUID) (CampaignDTO, error) {
	campaign, ok := s.byID[id]
	if !ok {
		return CampaignDTO{}, ErrCampaignNotFound
	}
	return campaign, nil
}

func TestGetCampaignDiff_selfReturnsEmptyRows_holdout(t *testing.T) {
	t.Parallel()
	campID := uuid.New()
	stub := &diffCampaignStub{byID: map[uuid.UUID]CampaignDTO{
		campID: {ID: campID.String(), Name: "Live", CustomerID: uuid.New().String()},
	}}
	h := &CampaignsHTTPHandlers{Campaigns: stub}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+campID.String()+"/diff?against="+campID.String(), http.NoBody)
	req.SetPathValue("id", campID.String())
	rec := httptest.NewRecorder()
	h.getCampaignDiff(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp CampaignDiffResponseDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Empty(t, resp.Rows)
}

func TestGetCampaignDiff_crossCustomerReturns404_holdout(t *testing.T) {
	t.Parallel()
	leftID := uuid.New()
	rightID := uuid.New()
	stub := &diffCampaignStub{byID: map[uuid.UUID]CampaignDTO{
		leftID:  {ID: leftID.String(), Name: "Left", CustomerID: uuid.New().String()},
		rightID: {ID: rightID.String(), Name: "Right", CustomerID: uuid.New().String()},
	}}
	h := &CampaignsHTTPHandlers{Campaigns: stub}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+leftID.String()+"/diff?against="+rightID.String(), http.NoBody)
	req.SetPathValue("id", leftID.String())
	rec := httptest.NewRecorder()
	h.getCampaignDiff(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetCampaignDiff_forbiddenAgainstReturns404_holdout(t *testing.T) {
	t.Parallel()
	leftID := uuid.New()
	rightID := uuid.New()
	customerID := uuid.New().String()
	stub := &diffCampaignStub{byID: map[uuid.UUID]CampaignDTO{
		leftID:  {ID: leftID.String(), Name: "Left", CustomerID: customerID},
		rightID: {ID: rightID.String(), Name: "Right", CustomerID: customerID},
	}}
	h := &CampaignsHTTPHandlers{
		Campaigns: stub,
		AuthorizeCampaignAccess: func(_ *http.Request, id uuid.UUID) error {
			if id == rightID {
				return ErrForbidden
			}
			return nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+leftID.String()+"/diff?against="+rightID.String(), http.NoBody)
	req.SetPathValue("id", leftID.String())
	rec := httptest.NewRecorder()
	h.getCampaignDiff(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetCampaignDiff_differentFieldsReturnRows(t *testing.T) {
	t.Parallel()
	leftID := uuid.New()
	rightID := uuid.New()
	customerID := uuid.New().String()
	stub := &diffCampaignStub{byID: map[uuid.UUID]CampaignDTO{
		leftID:  {ID: leftID.String(), Name: "Left", CustomerID: customerID, Status: "ACTIVE"},
		rightID: {ID: rightID.String(), Name: "Right", CustomerID: customerID, Status: "PAUSED"},
	}}
	h := &CampaignsHTTPHandlers{Campaigns: stub}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+leftID.String()+"/diff?against="+rightID.String(), http.NoBody)
	req.SetPathValue("id", leftID.String())
	rec := httptest.NewRecorder()
	h.getCampaignDiff(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp CampaignDiffResponseDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Rows)
}

func TestDataFreshnessFromClickHouse_nilQueryStale_holdout(t *testing.T) {
	t.Parallel()
	dto := reports.DataFreshnessFromClickHouse(context.Background(), nil)
	assert.True(t, dto.Stale)
}

func TestWriteBuyerFraudExportPreamble_signalsDegradedWhenStale_holdout(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	require.NoError(t, reports.WriteBuyerFraudExportPreamble(w, reports.DataFreshnessDTO{Stale: true}))
	w.Flush()
	content := buf.String()
	assert.Contains(t, content, "signals_degraded")
	assert.Contains(t, content, "true")
}

func TestWriteBuyerFraudExportPreamble_freshOmitsDegradedRow(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	require.NoError(t, reports.WriteBuyerFraudExportPreamble(w, reports.DataFreshnessDTO{Stale: false}))
	w.Flush()
	assert.NotContains(t, buf.String(), "# signals_degraded")
}
