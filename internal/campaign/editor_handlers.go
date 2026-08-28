package campaign

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type CampaignEditorSectionDTO struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Order      int    `json:"order"`
	Visible    bool   `json:"visible"`
	Complete   bool   `json:"complete"`
	IssueCount int    `json:"issue_count"`
	IssueTone  string `json:"issue_tone"`
}

type CampaignEditorFieldDTO struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Editable    bool   `json:"editable"`
	Required    bool   `json:"required"`
	ValueSource string `json:"value_source"`
}

type CampaignEditorShellDTO struct {
	CampaignID      string                      `json:"campaign_id"`
	Sections        []CampaignEditorSectionDTO  `json:"sections"`
	Fields          []CampaignEditorFieldDTO    `json:"fields,omitempty"`
	CompletionPct   int                         `json:"completion_pct"`
	AllowedActions  []string                    `json:"allowed_actions,omitempty"`
	ContextLinks    []CampaignContextLinkDTO    `json:"context_links,omitempty"`
	SchedulePreview *CampaignSchedulePreviewDTO `json:"schedule_preview,omitempty"`
	GeoSummary      *CampaignGeoSummaryDTO      `json:"geo_summary,omitempty"`
}

type CampaignValidateResponse struct {
	Valid                 bool                  `json:"valid"`
	FieldErrors           map[string]string     `json:"field_errors,omitempty"`
	Warnings              []string              `json:"warnings,omitempty"`
	Advisories            []CampaignAdvisoryDTO `json:"advisories,omitempty"`
	EstimatedOutboxEvents []string              `json:"estimated_outbox_events,omitempty"`
}

type CampaignAdvisoryDTO struct {
	Code            string  `json:"code"`
	Severity        string  `json:"severity"`
	Message         string  `json:"message"`
	RequiresConfirm bool    `json:"requires_confirm,omitempty"`
	ImpactLabel     string  `json:"impact_label,omitempty"`
	ProjectedPct    float64 `json:"projected_margin_pct,omitempty"`
	FloorPct        float64 `json:"floor_pct,omitempty"`
}

type CampaignIntegrationPanelDTO struct {
	OverallStatus      string                 `json:"overall_status"`
	OverallStatusLabel string                 `json:"overall_status_label"`
	OverallStatusTone  string                 `json:"overall_status_tone"`
	Rows               []IntegrationHealthRow `json:"rows"`
}

func (h *CampaignsHTTPHandlers) registerCampaignEditorRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	read := []string{"campaigns:read", authz.PermCampaignsReadMasked}
	write := []string{"campaigns:write"}
	mux.HandleFunc("GET /api/v1/campaigns/{id}/editor-shell", limit(perm(read, h.getCampaignEditorShell)))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/validate", limit(perm(write, h.postCampaignValidate)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/integration-panel", limit(perm(read, h.getCampaignIntegrationPanel)))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/flow/validate", limit(perm(write, h.postCampaignFlowValidate)))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/macro-preview", limit(perm(read, h.postCampaignMacroPreview)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/diff", limit(perm(read, h.getCampaignDiff)))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/clone/preview", limit(perm(write, h.postCampaignClonePreview)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/placement-block-suggestions", limit(perm(read, h.getPlacementBlockSuggestions)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/geo-summary", limit(perm(read, h.getCampaignGeoSummary)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/fraud/editor-summary", limit(perm(read, h.getCampaignFraudEditorSummary)))
	mux.HandleFunc("POST /api/v1/campaigns/bulk", limit(perm(write, h.postCampaignBulk)))
}

func (h *CampaignsHTTPHandlers) getCampaignEditorShell(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
	if !ok {
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	campaign, err := h.Campaigns.GetCampaign(r.Context(), campaignID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	health, _ := h.Campaigns.GetCampaignIntegrationHealth(r.Context(), campaignID)
	sections := buildCampaignEditorSections(r.Context(), h, campaign, health)
	complete, total := editorCompletionCounts(sections)
	schedulePreview := buildCampaignSchedulePreview(campaign, time.Now().UTC())
	geoSummary := buildCampaignGeoSummary(campaign, false)
	shell := CampaignEditorShellDTO{
		CampaignID:      campaign.ID,
		Sections:        sections,
		CompletionPct:   completionPercent(complete, total),
		AllowedActions:  campaign.AllowedActions,
		ContextLinks:    buildCampaignContextLinks(r.Context(), campaign),
		SchedulePreview: &schedulePreview,
		GeoSummary:      &geoSummary,
	}
	httpresponse.JSON(w, http.StatusOK, shell)
}

func (h *CampaignsHTTPHandlers) getCampaignGeoSummary(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
	if !ok {
		return
	}
	campaign, err := h.Campaigns.GetCampaign(r.Context(), campaignID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	expand := strings.TrimSpace(r.URL.Query().Get("expand")) == "1"
	httpresponse.JSON(w, http.StatusOK, buildCampaignGeoSummary(campaign, expand))
}

func (h *CampaignsHTTPHandlers) getCampaignFraudEditorSummary(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
	if !ok {
		return
	}
	if h.AllowFraudPreview != nil && !h.AllowFraudPreview(campaignID.String()) {
		httpresponse.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "fraud preview rate limit exceeded")
		return
	}
	campaign, err := h.Campaigns.GetCampaign(r.Context(), campaignID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	previewer, ok := h.Campaigns.(interface {
		PreviewCampaignFraud(context.Context, uuid.UUID, PreviewCampaignFraudRequest) (CampaignFraudPreviewDTO, error)
	})
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "fraud preview not configured")
		return
	}
	preview, err := previewer.PreviewCampaignFraud(r.Context(), campaignID, PreviewCampaignFraudRequest{})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, buildCampaignFraudEditorSummary(campaign, preview))
}

func (h *CampaignsHTTPHandlers) postCampaignValidate(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
	if !ok {
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req PatchCampaignRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	resp := validateCampaignPatch(r.Context(), campaignID, req)
	if h.Campaigns != nil {
		resp.Advisories = append(resp.Advisories, marginAdvisoryForCampaign(r.Context(), campaignID, h.Campaigns)...)
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (h *CampaignsHTTPHandlers) getCampaignIntegrationPanel(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
	if !ok {
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	health, err := h.Campaigns.GetCampaignIntegrationHealth(r.Context(), campaignID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	panel := buildCampaignIntegrationPanel(health)
	httpresponse.JSON(w, http.StatusOK, panel)
}

func buildCampaignEditorSections(ctx context.Context, handlers *CampaignsHTTPHandlers, campaign CampaignDTO, health IntegrationHealthDTO) []CampaignEditorSectionDTO {
	integrationIssues := integrationHealthIssueCount(health)
	sections := []CampaignEditorSectionDTO{
		{ID: "general", Title: "General", Order: 1, Visible: true, Complete: campaign.Name != "", IssueTone: "ok"},
		{ID: "budget_pacing", Title: "Budget and pacing", Order: 2, Visible: maskLevelFromContext(ctx) == authz.MaskFull, Complete: campaign.BudgetLimit != "", IssueTone: "ok"},
		{ID: "targeting", Title: "Targeting", Order: 3, Visible: true, Complete: len(campaign.TargetCountries) > 0, IssueTone: "ok"},
		{ID: "flow", Title: "Flow", Order: 4, Visible: true, Complete: campaign.FlowID != "", IssueTone: toneForBool(campaign.FlowID != "")},
		{ID: "tracking", Title: "Tracking", Order: 5, Visible: true, Complete: campaign.TrafficTemplateID != "" || len(campaign.ClickQueryParams) > 0, IssueTone: "ok"},
		{ID: "postbacks", Title: "Postbacks", Order: 6, Visible: true, Complete: true, IssueTone: "ok"},
		{ID: "fraud", Title: "Fraud", Order: 7, Visible: maskLevelFromContext(ctx) == authz.MaskFull, Complete: true, IssueTone: "ok"},
		{ID: "integrations", Title: "Integrations", Order: 8, Visible: true, Complete: integrationIssues == 0, IssueCount: integrationIssues, IssueTone: integrationHealthTone(health)},
	}
	if openRTB, _ := licenseFeatureAllowed(handlers, "openrtb"); openRTB && maskLevelFromContext(ctx) == authz.MaskFull {
		sections = append(sections, CampaignEditorSectionDTO{
			ID: "rtb", Title: "RTB", Order: 9, Visible: true, Complete: campaign.FlowID != "", IssueTone: toneForBool(campaign.FlowID != ""),
		})
	}
	visible := make([]CampaignEditorSectionDTO, 0, len(sections))
	for _, section := range sections {
		if section.Visible {
			visible = append(visible, section)
		}
	}
	return visible
}

func editorCompletionCounts(sections []CampaignEditorSectionDTO) (complete, total int) {
	for _, section := range sections {
		total++
		if section.Complete {
			complete++
		}
	}
	return complete, total
}

func completionPercent(complete, total int) int {
	if total == 0 {
		return 0
	}
	return complete * 100 / total
}

func toneForBool(ok bool) string {
	if ok {
		return "ok"
	}
	return "warn"
}

func licenseFeatureAllowed(handlers *CampaignsHTTPHandlers, featureKey string) (bool, string) {
	if handlers != nil && handlers.LicenseFeatureAllowed != nil {
		return handlers.LicenseFeatureAllowed(featureKey)
	}
	return true, ""
}

func validateCampaignPatch(ctx context.Context, campaignID uuid.UUID, req PatchCampaignRequest) CampaignValidateResponse {
	_ = ctx
	_ = campaignID
	errors := make(map[string]string)
	if req.PacingMode != nil {
		mode := strings.ToUpper(strings.TrimSpace(*req.PacingMode))
		if mode != "STANDARD" && mode != "ACCELERATED" && mode != "EVEN" {
			errors["pacing_mode"] = "invalid pacing mode"
		}
	}
	if snap, ok := authz.SnapshotFromContext(ctx); ok && snap.Mask == authz.MaskMasked {
		if req.BudgetLimit != nil || req.BudgetLimitMicro != nil || req.DailyBudgetMicro != nil {
			errors["budget_limit"] = "masked role cannot change budget"
		}
	}
	valid := len(errors) == 0
	return CampaignValidateResponse{Valid: valid, FieldErrors: errors}
}

func buildCampaignIntegrationPanel(health IntegrationHealthDTO) CampaignIntegrationPanelDTO {
	rows := make([]IntegrationHealthRow, len(health.Rows))
	for i := range health.Rows {
		rows[i] = health.Rows[i]
		rows[i].ActionID = health.Rows[i].Slug
		if rows[i].FixRoute != "" {
			rows[i].FixHintLabel = "Open fix route"
		}
	}
	tone := "ok"
	label := "Healthy"
	status := IntegrationHealthOK
	for _, row := range health.Rows {
		if row.Status == string(IntegrationHealthFail) {
			status = IntegrationHealthFail
			tone = "fail"
			label = "Action required"
			break
		}
		if row.Status == string(IntegrationHealthWarn) && status != IntegrationHealthFail {
			status = IntegrationHealthWarn
			tone = "warn"
			label = "Needs attention"
		}
	}
	return CampaignIntegrationPanelDTO{
		OverallStatus:      string(status),
		OverallStatusLabel: label,
		OverallStatusTone:  tone,
		Rows:               rows,
	}
}

func integrationHealthIssueCount(health IntegrationHealthDTO) int {
	count := 0
	for _, row := range health.Rows {
		if row.Status == string(IntegrationHealthFail) || row.Status == string(IntegrationHealthWarn) {
			count++
		}
	}
	return count
}

func integrationHealthTone(health IntegrationHealthDTO) string {
	if health.Summary == string(IntegrationHealthFail) {
		return "fail"
	}
	if health.Summary == string(IntegrationHealthWarn) {
		return "warn"
	}
	return "ok"
}

func filterAndSortCampaigns(items []CampaignDTO, q, sortField, order, pacingMode string) []CampaignDTO {
	filtered := make([]CampaignDTO, 0, len(items))
	q = strings.ToLower(strings.TrimSpace(q))
	for _, item := range items {
		if q != "" && !strings.Contains(strings.ToLower(item.Name), q) && !strings.Contains(strings.ToLower(item.ID), q) {
			continue
		}
		if pacingMode != "" && !strings.EqualFold(item.PacingMode, pacingMode) {
			continue
		}
		filtered = append(filtered, item)
	}
	if sortField == "" {
		sortField = "updated_at"
	}
	sort.Slice(filtered, func(i, j int) bool {
		less := false
		switch sortField {
		case "name":
			less = strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
		case "spend":
			less = filtered[i].CurrentSpend < filtered[j].CurrentSpend
		default:
			less = filtered[i].UpdatedAt < filtered[j].UpdatedAt
		}
		if order == "asc" {
			return less
		}
		return !less
	})
	return filtered
}
