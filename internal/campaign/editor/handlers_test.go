package editor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/reports"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCampaignContextLinks_buyerOmitsFraudEvidence_holdout(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Permissions: map[string]struct{}{authz.PermCampaignsReadMasked: {}},
		Mask:        authz.MaskMasked,
	})
	links := buildCampaignContextLinks(ctx, campaign.CampaignDTO{ID: "c1", CustomerID: "cust-1"})
	for _, link := range links {
		assert.NotEqual(t, "fraud-evidence-pack", link.ReportKey)
		assert.NotEqual(t, "filter-rejects", link.ReportKey)
	}
}

func TestBuildCampaignContextLinks_catalogKeysMatchReportCatalog(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Permissions: map[string]struct{}{"audit:read": {}, authz.PermCampaignsRead: {}},
		Mask:        authz.MaskFull,
	})
	links := buildCampaignContextLinks(ctx, campaign.CampaignDTO{ID: "c1", CustomerID: "cust-1"})
	keys := reports.LiveReportExportKeys()
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[key] = struct{}{}
	}
	for _, link := range links {
		if link.ReportKey == "" {
			continue
		}
		_, ok := keySet[link.ReportKey]
		assert.True(t, ok, "report key %q should be in catalog", link.ReportKey)
	}
}

func TestBuildCampaignSchedulePreview_dstSpringForward(t *testing.T) {
	t.Parallel()
	camp := campaign.CampaignDTO{
		Status:       "ACTIVE",
		Timezone:     "America/New_York",
		DaypartHours: []int16{10, 11, 12},
	}
	now := time.Date(2026, 3, 8, 14, 30, 0, 0, time.UTC)
	preview := buildCampaignSchedulePreview(camp, now)
	assert.Contains(t, preview.SummaryLabel, "America/New_York")
	assert.NotEmpty(t, preview.SummaryLabel)
}

func TestBuildCampaignEditorSections_openRTBHiddenWithoutLicense(t *testing.T) {
	t.Parallel()
	sections := buildCampaignEditorSections(context.Background(), nil, campaign.CampaignDTO{FlowID: "f1"}, campaign.IntegrationHealthDTO{})
	for _, section := range sections {
		assert.NotEqual(t, "rtb", section.ID)
	}
}

func TestBuildCampaignEditorSections_integrationIssueCount(t *testing.T) {
	t.Parallel()
	health := campaign.IntegrationHealthDTO{
		Summary: "warn",
		Rows: []campaign.IntegrationHealthRow{
			{Status: string(campaign.IntegrationHealthWarn)},
			{Status: string(campaign.IntegrationHealthOK)},
		},
	}
	sections := buildCampaignEditorSections(context.Background(), nil, campaign.CampaignDTO{}, health)
	var integrations CampaignEditorSectionDTO
	for _, section := range sections {
		if section.ID == "integrations" {
			integrations = section
			break
		}
	}
	assert.Equal(t, 1, integrations.IssueCount)
}

func TestValidateCampaignPatch_invalidPacingMode(t *testing.T) {
	t.Parallel()
	mode := "turbo"
	resp := validateCampaignPatch(context.Background(), uuid.New(), campaign.PatchCampaignRequest{PacingMode: &mode})
	assert.False(t, resp.Valid)
	assert.Contains(t, resp.FieldErrors, "pacing_mode")
}

func TestValidateCampaignPatch_maskedBudgetDenied(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{Mask: authz.MaskMasked})
	budget := "100.00"
	resp := validateCampaignPatch(ctx, uuid.New(), campaign.PatchCampaignRequest{BudgetLimit: &budget})
	assert.False(t, resp.Valid)
	assert.Contains(t, resp.FieldErrors, "budget_limit")
}

func TestPreviewCampaignMacros_resolvesRedirectMacros(t *testing.T) {
	t.Parallel()
	camp := campaign.CampaignDTO{
		ID:        "camp-1",
		TargetURL: "https://offer.example/lp?cid={click_id}&sub1={sub1}",
	}
	preview, err := previewCampaignMacros(camp, MacroPreviewRequestDTO{Sub1: "alpha"}, false)
	require.NoError(t, err)
	assert.Contains(t, preview.ResolvedClickURL, "preview-click-id")
	assert.Contains(t, preview.ResolvedClickURL, "alpha")
}

func TestPreviewCampaignMacros_unresolvedPresetMacro(t *testing.T) {
	t.Parallel()
	camp := campaign.CampaignDTO{
		ID:        "camp-1",
		TargetURL: "https://offer.example/?x={{adset.id}}",
	}
	preview, err := previewCampaignMacros(camp, MacroPreviewRequestDTO{}, false)
	require.NoError(t, err)
	require.NotEmpty(t, preview.UnresolvedMacros)
}

func TestPreviewCampaignMacros_resolvesCampaignId(t *testing.T) {
	t.Parallel()
	camp := campaign.CampaignDTO{
		ID:        "camp-1",
		TargetURL: "https://offer.example/click?cid={{campaign.id}}&sub1={{sub1}}",
	}
	preview, err := previewCampaignMacros(camp, MacroPreviewRequestDTO{Sub1: "alpha"}, false)
	require.NoError(t, err)
	assert.Contains(t, preview.ResolvedClickURL, "camp-1")
	assert.Contains(t, preview.ResolvedClickURL, "alpha")
}

func TestPreviewCampaignMacros_maskedRedactsOfferURL(t *testing.T) {
	t.Parallel()
	camp := campaign.CampaignDTO{ID: "camp-1", TargetURL: "https://secret.offer/track"}
	preview, err := previewCampaignMacros(camp, MacroPreviewRequestDTO{}, true)
	require.NoError(t, err)
	assert.Equal(t, "[redacted-offer-url]", preview.ResolvedClickURL)
}

func TestPostCampaignBulk_rejectsTooManyIDs(t *testing.T) {
	t.Parallel()
	ids := make([]string, bulkCampaignMaxSync+1)
	for i := range ids {
		ids[i] = uuid.New().String()
	}
	h := &campaign.CampaignsHTTPHandlers{Campaigns: &diffCampaignStub{}}
	mux := http.NewServeMux()
	RegisterRoutes(h, mux, func(next http.HandlerFunc) http.HandlerFunc { return next }, func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next })
	payload, err := json.Marshal(BulkCampaignRequestDTO{Action: "pause", CampaignIDs: ids})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/bulk-action", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPostCampaignBulk_archive(t *testing.T) {
	t.Parallel()
	campID := uuid.New()
	stub := &archiveBulkCampaignStub{}
	h := &campaign.CampaignsHTTPHandlers{Campaigns: stub}
	mux := http.NewServeMux()
	RegisterRoutes(h, mux, func(next http.HandlerFunc) http.HandlerFunc { return next }, func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next })
	payload, err := json.Marshal(BulkCampaignRequestDTO{Action: "archive", CampaignIDs: []string{campID.String()}})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/bulk-action", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp BulkCampaignResponseDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Results, 1)
	assert.True(t, resp.Results[0].OK)
	assert.Equal(t, campID.String(), resp.Results[0].ID)
	assert.Equal(t, []uuid.UUID{campID}, stub.archived)
	assert.Equal(t, "bulk_archive", stub.lastReason)
}

type archiveBulkCampaignStub struct {
	archived   []uuid.UUID
	lastReason string
}

func (s *archiveBulkCampaignStub) GetCampaign(context.Context, uuid.UUID) (campaign.CampaignDTO, error) {
	return campaign.CampaignDTO{}, campaign.ErrCampaignNotFound
}

func (s *archiveBulkCampaignStub) GetCampaignMargin(context.Context, uuid.UUID) (campaign.CampaignMarginDTO, error) {
	return campaign.CampaignMarginDTO{}, nil
}

func (s *archiveBulkCampaignStub) ListCampaigns(context.Context, uuid.UUID, string, int32, int32) ([]campaign.CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (s *archiveBulkCampaignStub) ListCampaignsFiltered(context.Context, campaign.ListCampaignsFilter) ([]campaign.CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (s *archiveBulkCampaignStub) CountCampaignStatusTotals(context.Context, campaign.ListCampaignsFilter, string, string) (campaign.CampaignStatusTotalsDTO, error) {
	return campaign.CampaignStatusTotalsDTO{}, nil
}

func (s *archiveBulkCampaignStub) AttachCampaignListMarginBreach(context.Context, []campaign.CampaignDTO) {
}

func (s *archiveBulkCampaignStub) PatchCampaign(context.Context, uuid.UUID, campaign.PatchCampaignRequest) (campaign.CampaignDTO, error) {
	return campaign.CampaignDTO{}, nil
}

func (s *archiveBulkCampaignStub) PublishCampaign(context.Context, uuid.UUID, bool) (campaign.CampaignDTO, error) {
	return campaign.CampaignDTO{}, nil
}

func (s *archiveBulkCampaignStub) EvaluateCampaignPublish(context.Context, uuid.UUID) (campaign.CampaignPublishCheckDTO, error) {
	return campaign.CampaignPublishCheckDTO{}, nil
}

func (s *archiveBulkCampaignStub) AssignCampaignOwner(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (s *archiveBulkCampaignStub) ListCampaignEvents(context.Context, uuid.UUID, int32, int32) ([]campaign.CampaignEventDTO, int64, error) {
	return nil, 0, nil
}

func (s *archiveBulkCampaignStub) BlockCampaignPlacement(context.Context, uuid.UUID, string) error {
	return nil
}

func (s *archiveBulkCampaignStub) CloneCampaign(context.Context, campaign.CloneCampaignSpec) (campaign.CloneCampaignResult, error) {
	return campaign.CloneCampaignResult{}, nil
}

func (s *archiveBulkCampaignStub) ExportCampaign(context.Context, uuid.UUID) (campaign.CampaignExportBundle, error) {
	return campaign.CampaignExportBundle{}, nil
}

func (s *archiveBulkCampaignStub) ImportCampaign(context.Context, campaign.ImportCampaignSpec) (campaign.ImportCampaignResult, error) {
	return campaign.ImportCampaignResult{}, nil
}

func (s *archiveBulkCampaignStub) ImportMigrationCampaigns(context.Context, campaign.ImportMigrationSpec) (campaign.ImportMigrationResult, error) {
	return campaign.ImportMigrationResult{}, nil
}

func (s *archiveBulkCampaignStub) GetCampaignIntegrationHealth(context.Context, uuid.UUID) (campaign.IntegrationHealthDTO, error) {
	return campaign.IntegrationHealthDTO{}, nil
}

func (s *archiveBulkCampaignStub) PauseCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func (s *archiveBulkCampaignStub) ResumeCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func (s *archiveBulkCampaignStub) ArchiveCampaign(_ context.Context, campaignID uuid.UUID, reason string) error {
	s.archived = append(s.archived, campaignID)
	s.lastReason = reason
	return nil
}

func TestPostCampaignBulk_unsupportedActionRejected(t *testing.T) {
	t.Parallel()
	h := &campaign.CampaignsHTTPHandlers{Campaigns: &diffCampaignStub{}}
	mux := http.NewServeMux()
	RegisterRoutes(h, mux, func(next http.HandlerFunc) http.HandlerFunc { return next }, func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next })
	body := `{"action":"adjust_budget_pct","campaign_ids":["` + uuid.New().String() + `"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/bulk-action", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPlacementBlockSuggestions_noAutoBlockRoute(t *testing.T) {
	t.Parallel()
	h := &campaign.CampaignsHTTPHandlers{}
	mux := http.NewServeMux()
	RegisterRoutes(h, mux, func(next http.HandlerFunc) http.HandlerFunc { return next }, func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next })
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/placements/ivt", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestGetCampaignFraudEditorSummary_rateLimitShared_holdout(t *testing.T) {
	t.Parallel()
	campaignID := uuid.New()
	h := &campaign.CampaignsHTTPHandlers{
		Campaigns: &campaignFraudPreviewStub{campaignID: campaignID},
		AllowFraudPreview: func(string) bool {
			return false
		},
	}
	mux := http.NewServeMux()
	RegisterRoutes(h, mux, func(next http.HandlerFunc) http.HandlerFunc { return next }, func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+campaignID.String()+"/fraud-editor", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestIntegrationPanel_forbiddenCampaign404(t *testing.T) {
	t.Parallel()
	campaignID := uuid.New()
	h := &campaign.CampaignsHTTPHandlers{
		Campaigns: &diffCampaignStub{},
		AuthorizeCampaignAccess: func(_ *http.Request, id uuid.UUID) error {
			if id == campaignID {
				return campaign.ErrCampaignNotFound
			}
			return nil
		},
		WriteServiceError: func(w http.ResponseWriter, err error) {
			if errors.Is(err, campaign.ErrCampaignNotFound) {
				httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
				return
			}
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		},
	}
	mux := http.NewServeMux()
	RegisterRoutes(h, mux, func(next http.HandlerFunc) http.HandlerFunc { return next }, func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+campaignID.String()+"/integration-panel", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

type campaignFraudPreviewStub struct {
	diffCampaignStub
	campaignID uuid.UUID
}

func (s *campaignFraudPreviewStub) PreviewCampaignFraud(_ context.Context, campaignID uuid.UUID, _ campaign.PreviewCampaignFraudRequest) (campaign.CampaignFraudPreviewDTO, error) {
	return campaign.CampaignFraudPreviewDTO{CampaignID: campaignID.String()}, nil
}

func (s *campaignFraudPreviewStub) GetCampaign(_ context.Context, campaignID uuid.UUID) (campaign.CampaignDTO, error) {
	return campaign.CampaignDTO{ID: campaignID.String()}, nil
}
