package reports

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"ad-event-processor/internal/automation"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type ReportRuleCreator interface {
	CreateRule(ctx context.Context, req automation.UpsertRuleRequest) (automation.RuleDTO, error)
}

type ReportAuditLogger func(ctx context.Context, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any)

type CreateReportRuleRequest struct {
	CustomerID     string            `json:"customer_id"`
	CampaignID     string            `json:"campaign_id"`
	ReportKey      string            `json:"report_key"`
	Name           string            `json:"name,omitempty"`
	Action         string            `json:"action"`
	Metric         string            `json:"metric,omitempty"`
	Operator       string            `json:"operator,omitempty"`
	Threshold      float64           `json:"threshold,omitempty"`
	WindowMinutes  int               `json:"window_minutes,omitempty"`
	FilterSnapshot map[string]string `json:"filter_snapshot"`
}

func (h *ReportsHTTPHandlers) registerReportRules(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	mux.HandleFunc("POST /api/v1/reports/rules", limit(perm("campaigns:write", h.postReportRule)))
}

func (h *ReportsHTTPHandlers) postReportRule(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.ReportRuleCreator == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "report rules unavailable")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[CreateReportRuleRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	customerID, err := uuid.Parse(strings.TrimSpace(req.CustomerID))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	if h.AuthorizeCustomerAccess != nil {
		if err := h.AuthorizeCustomerAccess(r, customerID.String()); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	campaignID, err := uuid.Parse(strings.TrimSpace(req.CampaignID))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	automationReq, meta, err := BuildAutomationRuleFromReport(req)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	rule, err := h.ReportRuleCreator.CreateRule(r.Context(), automationReq)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if h.ReportAuditLog != nil {
		adminID := uuid.Nil
		if u, ok := authz.GetUser(r.Context()); ok {
			adminID = u.UserID
		}
		ruleID, _ := uuid.Parse(rule.ID)
		target := ruleID
		h.ReportAuditLog(r.Context(), adminID, "CREATE_REPORT_AUTOMATION_RULE", "automation_rule", &target, map[string]any{
			"report_key":  req.ReportKey,
			"action":      req.Action,
			"campaign_id": campaignID.String(),
		}, meta)
	}
	httpresponse.JSON(w, http.StatusCreated, rule)
}

func BuildAutomationRuleFromReport(req CreateReportRuleRequest) (automation.UpsertRuleRequest, map[string]any, error) {
	reportKey := strings.TrimSpace(req.ReportKey)
	if reportKey == "" {
		return automation.UpsertRuleRequest{}, nil, fmt.Errorf("report_key is required")
	}
	action := strings.TrimSpace(req.Action)
	if action == "" {
		return automation.UpsertRuleRequest{}, nil, fmt.Errorf("action is required")
	}
	if len(req.FilterSnapshot) == 0 {
		return automation.UpsertRuleRequest{}, nil, fmt.Errorf("filter_snapshot is required")
	}
	metric := strings.TrimSpace(req.Metric)
	operator := strings.TrimSpace(req.Operator)
	threshold := req.Threshold
	if metric == "" || operator == "" || threshold == 0 {
		defaultMetric, defaultOperator, defaultThreshold := defaultReportRuleMetric(reportKey, action)
		if metric == "" {
			metric = defaultMetric
		}
		if operator == "" {
			operator = defaultOperator
		}
		if threshold == 0 {
			threshold = defaultThreshold
		}
	}
	var actions []automation.Action
	var groupBy string
	switch action {
	case automation.ActionPauseCampaign:
		actions = []automation.Action{{Type: automation.ActionPauseCampaign}}
		groupBy = automation.GroupByCampaign
	case automation.ActionBlacklistPlacement:
		actions = []automation.Action{{Type: automation.ActionBlacklistPlacement}}
		groupBy = automation.GroupByPlacement
	default:
		return automation.UpsertRuleRequest{}, nil, fmt.Errorf("unsupported action %q", action)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = fmt.Sprintf("Report rule: %s", reportKey)
	}
	snapshotJSON, _ := json.Marshal(req.FilterSnapshot)
	meta := map[string]any{
		"report_key":      reportKey,
		"filter_snapshot": req.FilterSnapshot,
		"snapshot_json":   string(snapshotJSON),
	}
	return automation.UpsertRuleRequest{
		CustomerID:          strings.TrimSpace(req.CustomerID),
		CampaignID:          strings.TrimSpace(req.CampaignID),
		Name:                name,
		Metric:              metric,
		Operator:            operator,
		Threshold:           threshold,
		WindowMinutes:       req.WindowMinutes,
		GroupBy:             groupBy,
		Actions:             actions,
		Enabled:             true,
		EvalIntervalMinutes: 15,
		CooldownMinutes:     60,
	}, meta, nil
}

func defaultReportRuleMetric(reportKey, action string) (metric, operator string, threshold float64) {
	switch {
	case strings.Contains(reportKey, "fraud"):
		return "fraud_reject_rate", "gt", 25
	case action == automation.ActionPauseCampaign:
		return "spend_micro", "gt", 50_000_000
	default:
		return "roi_pct", "lt", -40
	}
}
