package campaign

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	links := buildCampaignContextLinks(ctx, CampaignDTO{ID: "c1", CustomerID: "cust-1"})
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
	links := buildCampaignContextLinks(ctx, CampaignDTO{ID: "c1", CustomerID: "cust-1"})
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
	campaign := CampaignDTO{
		Status:       "ACTIVE",
		Timezone:     "America/New_York",
		DaypartHours: []int16{10, 11, 12},
	}
	now := time.Date(2026, 3, 8, 14, 30, 0, 0, time.UTC)
	preview := buildCampaignSchedulePreview(campaign, now)
	assert.Contains(t, preview.SummaryLabel, "America/New_York")
	assert.NotEmpty(t, preview.SummaryLabel)
}

func TestPostCampaignBulk_rejectsTooManyIDs(t *testing.T) {
	t.Parallel()
	ids := make([]string, bulkCampaignMaxSync+1)
	for i := range ids {
		ids[i] = uuid.New().String()
	}
	h := &CampaignsHTTPHandlers{Campaigns: &patchRevisionCampaignStub{}}
	mux := http.NewServeMux()
	h.registerCampaignEditorRoutes(mux, func(next http.HandlerFunc) http.HandlerFunc { return next }, func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next })
	payload, err := json.Marshal(BulkCampaignRequestDTO{Action: "pause", CampaignIDs: ids})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/bulk", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBuildCampaignGeoSummary_emptyRulesNeutralLabel(t *testing.T) {
	t.Parallel()
	out := buildCampaignGeoSummary(CampaignDTO{}, false)
	assert.Equal(t, "any country", out.IncludedLabel)
	assert.Equal(t, "none", out.ExcludedLabel)
}

func TestBuildCampaignGeoSummary_expandCapsAt50_holdout(t *testing.T) {
	t.Parallel()
	codes := make([]string, 60)
	for i := range codes {
		codes[i] = fmt.Sprintf("C%02d", i)
	}
	out := buildCampaignGeoSummary(CampaignDTO{TargetCountries: codes}, true)
	assert.True(t, out.Truncated)
	assert.Len(t, out.Expanded, campaignGeoExpandMaxRows)
}

func TestBuildCampaignEventTimeline_masksActor_holdout(t *testing.T) {
	t.Parallel()
	timeline := buildCampaignEventTimeline([]CampaignEventDTO{{
		EventType: "click",
		UserID:    "user-secret-id",
		CreatedAt: "2026-08-27T12:00:00Z",
	}}, true)
	require.Len(t, timeline.Days, 1)
	require.Len(t, timeline.Days[0].Events, 1)
	assert.Equal(t, "us***id", timeline.Days[0].Events[0].ActorLabel)
}

func TestBuildCampaignEditorSections_openRTBHiddenWithoutLicense(t *testing.T) {
	t.Parallel()
	sections := buildCampaignEditorSections(context.Background(), nil, CampaignDTO{FlowID: "f1"}, IntegrationHealthDTO{})
	for _, section := range sections {
		assert.NotEqual(t, "rtb", section.ID)
	}
}

func TestBuildCampaignEditorSections_integrationIssueCount(t *testing.T) {
	t.Parallel()
	health := IntegrationHealthDTO{
		Summary: "warn",
		Rows: []IntegrationHealthRow{
			{Status: string(IntegrationHealthWarn)},
			{Status: string(IntegrationHealthOK)},
		},
	}
	sections := buildCampaignEditorSections(context.Background(), nil, CampaignDTO{}, health)
	var integrations CampaignEditorSectionDTO
	for _, section := range sections {
		if section.ID == "integrations" {
			integrations = section
			break
		}
	}
	assert.Equal(t, 1, integrations.IssueCount)
}

func TestAttachInvalidSpendKPI_noDoubleCount_holdout(t *testing.T) {
	t.Parallel()
	out := reports.BuildCustomerFraudOverview(100, 40, 10, DataFreshnessDTO{})
	reports.AttachInvalidSpendKPI(&out, 40, 10, 100, 1_000_000, 0.95)
	assert.Equal(t, int64(500_000), out.InvalidSpendMicros)
}

func TestAttachInvalidSpendKPI_lowCoverageDisclaimer(t *testing.T) {
	t.Parallel()
	out := reports.BuildCustomerFraudOverview(100, 10, 5, DataFreshnessDTO{})
	reports.AttachInvalidSpendKPI(&out, 10, 5, 100, 1_000_000, 0.5)
	assert.Contains(t, out.Disclaimer, "90%")
}

func TestAllowedActionsDirectRoute403_holdout(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Permissions: map[string]struct{}{authz.PermCampaignsReadMasked: {}, authz.PermCampaignsPause: {}},
		Mask:        authz.MaskMasked,
	})
	actions, denied := computeCampaignAllowedActions(ctx, "ACTIVE")
	assert.Contains(t, actions, "pause")
	assert.NotContains(t, actions, "edit_fraud")
	assert.Equal(t, "requires_campaigns_write", denied["edit_fraud"])
	budget := "200.00"
	resp := validateCampaignPatch(ctx, uuid.New(), PatchCampaignRequest{BudgetLimit: &budget})
	assert.False(t, resp.Valid)
}

func TestPlacementBlockSuggestions_noAutoBlockRoute(t *testing.T) {
	t.Parallel()
	h := &CampaignsHTTPHandlers{}
	mux := http.NewServeMux()
	h.registerCampaignEditorRoutes(mux, func(next http.HandlerFunc) http.HandlerFunc { return next }, func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next })
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/"+uuid.New().String()+"/placement-block-suggestions", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestBuildCampaignFraudEditorSummary_noCorpusPaths(t *testing.T) {
	t.Parallel()
	summary := buildCampaignFraudEditorSummary(CampaignDTO{ID: "c1", CustomerID: "cust-1"}, CampaignFraudPreviewDTO{
		ByTier: FraudPreviewTierCountsDTO{Suspect: 2},
	})
	for _, card := range summary.Cards {
		assert.NotContains(t, card.ReportHref, "corpus")
		assert.NotContains(t, card.TitleLabel, "registry")
	}
}

func TestPostCampaignBulk_unsupportedActionRejected(t *testing.T) {
	t.Parallel()
	h := &CampaignsHTTPHandlers{
		Campaigns: &patchRevisionCampaignStub{},
		WriteServiceError: func(w http.ResponseWriter, err error) {
			status, code, msg := mapServiceError(err)
			httpresponse.Error(w, status, code, msg)
		},
	}
	mux := http.NewServeMux()
	h.registerCampaignEditorRoutes(mux, func(next http.HandlerFunc) http.HandlerFunc { return next }, func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next })
	body := `{"action":"adjust_budget_pct","campaign_ids":["` + uuid.New().String() + `"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetCampaignFraudEditorSummary_rateLimitShared_holdout(t *testing.T) {
	t.Parallel()
	campaignID := uuid.New()
	h := &CampaignsHTTPHandlers{
		Campaigns: &campaignFraudPreviewStub{campaignID: campaignID},
		AllowFraudPreview: func(string) bool {
			return false
		},
		WriteServiceError: func(w http.ResponseWriter, err error) {
			status, code, msg := mapServiceError(err)
			httpresponse.Error(w, status, code, msg)
		},
	}
	mux := http.NewServeMux()
	h.registerCampaignEditorRoutes(mux, func(next http.HandlerFunc) http.HandlerFunc { return next }, func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+campaignID.String()+"/fraud/editor-summary", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

type campaignFraudPreviewStub struct {
	patchRevisionCampaignStub
	campaignID uuid.UUID
}

func (s *campaignFraudPreviewStub) PreviewCampaignFraud(_ context.Context, campaignID uuid.UUID, _ PreviewCampaignFraudRequest) (CampaignFraudPreviewDTO, error) {
	return CampaignFraudPreviewDTO{CampaignID: campaignID.String()}, nil
}

func (s *campaignFraudPreviewStub) GetCampaign(_ context.Context, campaignID uuid.UUID) (CampaignDTO, error) {
	return CampaignDTO{ID: campaignID.String()}, nil
}
