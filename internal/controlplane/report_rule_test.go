package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/automation"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type stubReportRuleCreator struct {
	last automation.UpsertRuleRequest
}

func (s *stubReportRuleCreator) CreateRule(_ context.Context, req automation.UpsertRuleRequest) (automation.RuleDTO, error) {
	s.last = req
	return automation.RuleDTO{
		ID:         "00000000-0000-4000-8000-000000000099",
		CustomerID: req.CustomerID,
		Name:       req.Name,
		Metric:     req.Metric,
		Enabled:    true,
	}, nil
}

func TestReportRule_postCreatesAutomationRule_holdout(t *testing.T) {
	creator := &stubReportRuleCreator{}
	h := &reports.ReportsHTTPHandlers{
		ReportRuleCreator: creator,
		ApplyRateLimit:    func(next http.HandlerFunc) http.HandlerFunc { return next },
		RequirePermission: func(_ string, next http.HandlerFunc) http.HandlerFunc { return next },
	}
	mux := http.NewServeMux()
	h.Register(mux)

	body := map[string]any{
		"customer_id": "00000000-0000-4000-8000-000000000001",
		"campaign_id": "00000000-0000-4000-8000-000000000010",
		"report_key":  "fraud-breakdown",
		"action":      automation.ActionBlacklistPlacement,
		"filter_snapshot": map[string]string{
			"placement_id": "slot-42",
			"from":         "2026-09-01",
		},
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/rules", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, automation.GroupByPlacement, creator.last.GroupBy)
	require.Equal(t, automation.ActionBlacklistPlacement, creator.last.Actions[0].Type)
}

func TestReportRule_auditHookReceivesMetadata(t *testing.T) {
	creator := &stubReportRuleCreator{}
	var audited bool
	h := &reports.ReportsHTTPHandlers{
		ReportRuleCreator: creator,
		ReportAuditLog: func(_ context.Context, _ uuid.UUID, action, _ string, _ *uuid.UUID, _, metadata any) {
			if action == "CREATE_REPORT_AUTOMATION_RULE" && metadata != nil {
				audited = true
			}
		},
		ApplyRateLimit:    func(next http.HandlerFunc) http.HandlerFunc { return next },
		RequirePermission: func(_ string, next http.HandlerFunc) http.HandlerFunc { return next },
	}
	mux := http.NewServeMux()
	h.Register(mux)

	body := map[string]any{
		"customer_id": "00000000-0000-4000-8000-000000000001",
		"campaign_id": "00000000-0000-4000-8000-000000000010",
		"report_key":  "true-roi",
		"action":      automation.ActionPauseCampaign,
		"filter_snapshot": map[string]string{
			"from": "2026-09-01",
		},
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/rules", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.True(t, audited)
}
