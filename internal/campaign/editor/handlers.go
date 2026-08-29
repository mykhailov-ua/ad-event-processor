package editor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/internal/campaign"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/reports"
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
	CampaignID      string                          `json:"campaign_id"`
	Sections        []CampaignEditorSectionDTO      `json:"sections"`
	Fields          []CampaignEditorFieldDTO        `json:"fields,omitempty"`
	CompletionPct   int                             `json:"completion_pct"`
	AllowedActions  []string                        `json:"allowed_actions,omitempty"`
	ContextLinks    []CampaignContextLinkDTO        `json:"context_links,omitempty"`
	SchedulePreview *CampaignSchedulePreviewDTO     `json:"schedule_preview,omitempty"`
	GeoSummary      *campaign.CampaignGeoSummaryDTO `json:"geo_summary,omitempty"`
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
	OverallStatus      string                          `json:"overall_status"`
	OverallStatusLabel string                          `json:"overall_status_label"`
	OverallStatusTone  string                          `json:"overall_status_tone"`
	Rows               []campaign.IntegrationHealthRow `json:"rows"`
}

func getCampaignEditorShell(h *campaign.CampaignsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	camp, err := h.Campaigns.GetCampaign(r.Context(), campaignID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	health, _ := h.Campaigns.GetCampaignIntegrationHealth(r.Context(), campaignID)
	sections := buildCampaignEditorSections(r.Context(), h, camp, health)
	complete, total := editorCompletionCounts(sections)
	schedulePreview := buildCampaignSchedulePreview(camp, time.Now().UTC())
	geoSummary := campaign.BuildCampaignGeoSummary(camp, false)
	shell := CampaignEditorShellDTO{
		CampaignID:      camp.ID,
		Sections:        sections,
		CompletionPct:   completionPercent(complete, total),
		AllowedActions:  camp.AllowedActions,
		ContextLinks:    buildCampaignContextLinks(r.Context(), camp),
		SchedulePreview: &schedulePreview,
		GeoSummary:      &geoSummary,
	}
	httpresponse.JSON(w, http.StatusOK, shell)
}

func getCampaignGeoSummary(h *campaign.CampaignsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	camp, err := h.Campaigns.GetCampaign(r.Context(), campaignID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	expand := strings.TrimSpace(r.URL.Query().Get("expand")) == "1"
	httpresponse.JSON(w, http.StatusOK, campaign.BuildCampaignGeoSummary(camp, expand))
}

func getCampaignFraudEditorSummary(h *campaign.CampaignsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	if h.AllowFraudPreview != nil && !h.AllowFraudPreview(campaignID.String()) {
		httpresponse.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "fraud preview rate limit exceeded")
		return
	}
	camp, err := h.Campaigns.GetCampaign(r.Context(), campaignID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	previewer, ok := h.Campaigns.(interface {
		PreviewCampaignFraud(context.Context, uuid.UUID, campaign.PreviewCampaignFraudRequest) (campaign.CampaignFraudPreviewDTO, error)
	})
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "fraud preview not configured")
		return
	}
	preview, err := previewer.PreviewCampaignFraud(r.Context(), campaignID, campaign.PreviewCampaignFraudRequest{})
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, campaign.BuildCampaignFraudEditorSummary(camp, preview))
}

func postCampaignValidate(h *campaign.CampaignsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req campaign.PatchCampaignRequest
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

func getCampaignIntegrationPanel(h *campaign.CampaignsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	health, err := h.Campaigns.GetCampaignIntegrationHealth(r.Context(), campaignID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	panel := buildCampaignIntegrationPanel(health)
	httpresponse.JSON(w, http.StatusOK, panel)
}

func buildCampaignEditorSections(ctx context.Context, handlers *campaign.CampaignsHTTPHandlers, camp campaign.CampaignDTO, health campaign.IntegrationHealthDTO) []CampaignEditorSectionDTO {
	integrationIssues := integrationHealthIssueCount(health)
	sections := []CampaignEditorSectionDTO{
		{ID: "general", Title: "General", Order: 1, Visible: true, Complete: camp.Name != "", IssueTone: "ok"},
		{ID: "budget_pacing", Title: "Budget and pacing", Order: 2, Visible: campaign.MaskLevelFromContext(ctx) == authz.MaskFull, Complete: camp.BudgetLimit != "", IssueTone: "ok"},
		{ID: "targeting", Title: "Targeting", Order: 3, Visible: true, Complete: len(camp.TargetCountries) > 0, IssueTone: "ok"},
		{ID: "flow", Title: "Flow", Order: 4, Visible: true, Complete: camp.FlowID != "", IssueTone: toneForBool(camp.FlowID != "")},
		{ID: "tracking", Title: "Tracking", Order: 5, Visible: true, Complete: camp.TrafficTemplateID != "" || len(camp.ClickQueryParams) > 0, IssueTone: "ok"},
		{ID: "postbacks", Title: "Postbacks", Order: 6, Visible: true, Complete: true, IssueTone: "ok"},
		{ID: "fraud", Title: "Fraud", Order: 7, Visible: campaign.MaskLevelFromContext(ctx) == authz.MaskFull, Complete: true, IssueTone: "ok"},
		{ID: "integrations", Title: "Integrations", Order: 8, Visible: true, Complete: integrationIssues == 0, IssueCount: integrationIssues, IssueTone: integrationHealthTone(health)},
	}
	if openRTB, _ := licenseFeatureAllowed(handlers, "openrtb"); openRTB && campaign.MaskLevelFromContext(ctx) == authz.MaskFull {
		sections = append(sections, CampaignEditorSectionDTO{
			ID: "rtb", Title: "RTB", Order: 9, Visible: true, Complete: camp.FlowID != "", IssueTone: toneForBool(camp.FlowID != ""),
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

func licenseFeatureAllowed(handlers *campaign.CampaignsHTTPHandlers, featureKey string) (bool, string) {
	if handlers != nil && handlers.LicenseFeatureAllowed != nil {
		return handlers.LicenseFeatureAllowed(featureKey)
	}
	return true, ""
}

func validateCampaignPatch(ctx context.Context, campaignID uuid.UUID, req campaign.PatchCampaignRequest) CampaignValidateResponse {
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

func buildCampaignIntegrationPanel(health campaign.IntegrationHealthDTO) CampaignIntegrationPanelDTO {
	rows := make([]campaign.IntegrationHealthRow, len(health.Rows))
	for i := range health.Rows {
		rows[i] = health.Rows[i]
		rows[i].ActionID = health.Rows[i].Slug
		if rows[i].FixRoute != "" {
			rows[i].FixHintLabel = "Open fix route"
		}
	}
	tone := "ok"
	label := "Healthy"
	status := campaign.IntegrationHealthOK
	for _, row := range health.Rows {
		if row.Status == string(campaign.IntegrationHealthFail) {
			status = campaign.IntegrationHealthFail
			tone = "fail"
			label = "Action required"
			break
		}
		if row.Status == string(campaign.IntegrationHealthWarn) && status != campaign.IntegrationHealthFail {
			status = campaign.IntegrationHealthWarn
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

func integrationHealthIssueCount(health campaign.IntegrationHealthDTO) int {
	count := 0
	for _, row := range health.Rows {
		if row.Status == string(campaign.IntegrationHealthFail) || row.Status == string(campaign.IntegrationHealthWarn) {
			count++
		}
	}
	return count
}

func integrationHealthTone(health campaign.IntegrationHealthDTO) string {
	if health.Summary == string(campaign.IntegrationHealthFail) {
		return "fail"
	}
	if health.Summary == string(campaign.IntegrationHealthWarn) {
		return "warn"
	}
	return "ok"
}

type MacroPreviewRequestDTO struct {
	Sub1    string `json:"sub1,omitempty"`
	Country string `json:"country,omitempty"`
	ClickID string `json:"click_id,omitempty"`
	UserID  string `json:"user_id,omitempty"`
	FBCLID  string `json:"fbclid,omitempty"`
	GCLID   string `json:"gclid,omitempty"`
	TTCLID  string `json:"ttclid,omitempty"`
}

type MacroPreviewResponseDTO struct {
	ResolvedClickURL    string   `json:"resolved_click_url,omitempty"`
	ResolvedPostbackURL string   `json:"resolved_postback_url,omitempty"`
	UnresolvedMacros    []string `json:"unresolved_macros,omitempty"`
	Warnings            []string `json:"warnings,omitempty"`
}

func previewCampaignMacros(camp campaign.CampaignDTO, req MacroPreviewRequestDTO, masked bool) (MacroPreviewResponseDTO, error) {
	baseURL := camp.TargetURL
	if masked {
		baseURL = "[redacted-offer-url]"
	}
	if strings.TrimSpace(baseURL) == "" {
		return MacroPreviewResponseDTO{}, campaign.ErrValidationf("target_url is required for macro preview")
	}
	macroCtx := campaign.PreviewContext(camp.ID, campaign.PreviewRequest{
		Sub1:    req.Sub1,
		Country: req.Country,
		ClickID: req.ClickID,
		UserID:  req.UserID,
		FBCLID:  req.FBCLID,
		GCLID:   req.GCLID,
		TTCLID:  req.TTCLID,
	})
	resolved, unresolved := campaign.Expand(baseURL, macroCtx)
	if params := camp.ClickQueryParams; len(params) > 0 {
		var parts []string
		for k, v := range params {
			expanded, paramUnresolved := campaign.Expand(v, macroCtx)
			unresolved = append(unresolved, paramUnresolved...)
			parts = append(parts, k+"="+expanded)
		}
		if len(parts) > 0 && !strings.Contains(resolved, "?") {
			resolved += "?" + strings.Join(parts, "&")
		}
	}
	var warnings []string
	if strings.HasPrefix(strings.ToLower(resolved), "http://") {
		warnings = append(warnings, "click url uses http")
	}
	return MacroPreviewResponseDTO{
		ResolvedClickURL: resolved,
		UnresolvedMacros: unresolved,
		Warnings:         warnings,
	}, nil
}

type CampaignContextLinkDTO struct {
	Href                string   `json:"href"`
	Label               string   `json:"label"`
	ReportKey           string   `json:"report_key,omitempty"`
	DashboardKey        string   `json:"dashboard_key,omitempty"`
	DefaultRange        string   `json:"default_range,omitempty"`
	RequiredPermissions []string `json:"required_permissions,omitempty"`
}

type CampaignSchedulePreviewDTO struct {
	SummaryLabel          string `json:"summary_label"`
	CurrentlyActive       bool   `json:"currently_active"`
	NextActivationAt      string `json:"next_activation_at,omitempty"`
	NextDeactivationAt    string `json:"next_deactivation_at,omitempty"`
	ScheduleSparseWarning bool   `json:"schedule_sparse_warning,omitempty"`
}

func buildCampaignContextLinks(ctx context.Context, camp campaign.CampaignDTO) []CampaignContextLinkDTO {
	links := []CampaignContextLinkDTO{
		{Href: fmt.Sprintf("/reports/campaign-overview?campaign_id=%s&customer_id=%s", camp.ID, camp.CustomerID), Label: "Campaign overview", ReportKey: "campaign-overview", DefaultRange: "7d", RequiredPermissions: []string{"campaigns:read"}},
		{Href: fmt.Sprintf("/reports/pacing-drift?campaign_id=%s&customer_id=%s", camp.ID, camp.CustomerID), Label: "Pacing drift", ReportKey: "pacing-drift", DefaultRange: "7d", RequiredPermissions: []string{"campaigns:read"}},
		{Href: fmt.Sprintf("/reports/wire-signal-breakdown?campaign_id=%s&customer_id=%s", camp.ID, camp.CustomerID), Label: "Wire signals", ReportKey: "wire-signal-breakdown", DefaultRange: "7d", RequiredPermissions: reports.ReportPermsFraudCustomer()},
	}
	if snap, ok := authz.SnapshotFromContext(ctx); ok && snap.Has("audit:read") && snap.Mask != authz.MaskMasked {
		links = append(links, CampaignContextLinkDTO{
			Href:  fmt.Sprintf("/reports/filter-rejects?campaign_id=%s&customer_id=%s", camp.ID, camp.CustomerID),
			Label: "Filter rejects", ReportKey: "filter-rejects", DefaultRange: "24h", RequiredPermissions: []string{"audit:read"},
		})
	}
	return links
}

func buildCampaignSchedulePreview(camp campaign.CampaignDTO, now time.Time) CampaignSchedulePreviewDTO {
	tz := strings.TrimSpace(camp.Timezone)
	if tz == "" {
		tz = "UTC"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	localNow := now.In(loc)
	active := true
	var startAt, endAt *time.Time
	if camp.StartAt != "" {
		if parsed, err := time.Parse(time.RFC3339, camp.StartAt); err == nil {
			startAt = &parsed
			if localNow.Before(parsed.In(loc)) {
				active = false
			}
		}
	}
	if camp.EndAt != "" {
		if parsed, err := time.Parse(time.RFC3339, camp.EndAt); err == nil {
			endAt = &parsed
			if !localNow.Before(parsed.In(loc)) {
				active = false
			}
		}
	}
	hours := len(camp.DaypartHours)
	if hours > 0 {
		active = false
		for _, h := range camp.DaypartHours {
			if int(h) == localNow.Hour() {
				active = true
				break
			}
		}
	}
	label := "Active around the clock"
	if hours > 0 {
		label = fmt.Sprintf("Active %d hour slots in %s", hours, tz)
	}
	if startAt != nil || endAt != nil {
		label = "Scheduled delivery window configured"
	}
	out := CampaignSchedulePreviewDTO{
		SummaryLabel:    label,
		CurrentlyActive: active && strings.EqualFold(camp.Status, "ACTIVE"),
	}
	if hours > 0 && hours < 84 {
		out.ScheduleSparseWarning = float64(hours)/168.0 < 0.5
	}
	if startAt != nil && localNow.Before(startAt.In(loc)) {
		out.NextActivationAt = startAt.UTC().Format(time.RFC3339)
	}
	if endAt != nil && localNow.Before(endAt.In(loc)) {
		out.NextDeactivationAt = endAt.UTC().Format(time.RFC3339)
	}
	return out
}

type CampaignDiffRowDTO struct {
	Path         string `json:"path"`
	Label        string `json:"label"`
	LeftDisplay  string `json:"left_display"`
	RightDisplay string `json:"right_display"`
	Severity     string `json:"severity"`
}

type CampaignDiffResponseDTO struct {
	Rows      []CampaignDiffRowDTO `json:"rows"`
	Truncated bool                 `json:"truncated,omitempty"`
}

const maxCampaignDiffRows = 200

func diffCampaignDTOs(left, right campaign.CampaignDTO, masked bool) CampaignDiffResponseDTO {
	fields := []struct {
		path  string
		label string
		left  string
		right string
	}{
		{"name", "Name", left.Name, right.Name},
		{"status", "Status", left.Status, right.Status},
		{"pacing_mode", "Pacing mode", left.PacingMode, right.PacingMode},
		{"budget_limit", "Budget", displayMoney(left.BudgetLimit, left.BudgetLimitDisplay, masked), displayMoney(right.BudgetLimit, right.BudgetLimitDisplay, masked)},
		{"target_url", "Target URL", redactURL(left.TargetURL, masked), redactURL(right.TargetURL, masked)},
		{"flow_id", "Flow", left.FlowID, right.FlowID},
	}
	var rows []CampaignDiffRowDTO
	for _, field := range fields {
		if field.left == field.right {
			continue
		}
		severity := "change"
		if field.left == "" {
			severity = "add"
		}
		if field.right == "" {
			severity = "remove"
		}
		rows = append(rows, CampaignDiffRowDTO{
			Path: field.path, Label: field.label,
			LeftDisplay: field.left, RightDisplay: field.right, Severity: severity,
		})
	}
	truncated := len(rows) > maxCampaignDiffRows
	if truncated {
		rows = rows[:maxCampaignDiffRows]
	}
	return CampaignDiffResponseDTO{Rows: rows, Truncated: truncated}
}

func displayMoney(raw, display string, masked bool) string {
	if masked && display != "" {
		return display
	}
	return raw
}

func redactURL(raw string, masked bool) string {
	if masked && raw != "" {
		return "[redacted]"
	}
	return raw
}

type CloneCampaignPreviewDTO struct {
	SourceID    string                        `json:"source_id"`
	Name        string                        `json:"name"`
	WouldCreate campaign.CloneCampaignOptions `json:"would_create"`
}

func previewCloneCampaign(camp campaign.CampaignDTO, req campaign.CloneCampaignHTTPRequest) CloneCampaignPreviewDTO {
	opts := campaign.NormalizedCloneOptions(req.Options)
	name := campaign.CloneCampaignName(camp.Name, req.NamePrefix, req.NameSuffix)
	return CloneCampaignPreviewDTO{
		SourceID:    camp.ID,
		Name:        name,
		WouldCreate: opts,
	}
}

type BulkCampaignRequestDTO struct {
	Action      string   `json:"action"`
	CampaignIDs []string `json:"campaign_ids"`
	PacingMode  string   `json:"pacing_mode,omitempty"`
}

type BulkCampaignResultRowDTO struct {
	ID        string `json:"id"`
	OK        bool   `json:"ok"`
	ErrorCode string `json:"error_code,omitempty"`
}

type BulkCampaignResponseDTO struct {
	Results []BulkCampaignResultRowDTO `json:"results"`
}

const bulkCampaignMaxSync = 50

func marginAdvisoryForCampaign(ctx context.Context, campaignID uuid.UUID, reader campaign.CampaignReader) []CampaignAdvisoryDTO {
	_ = ctx
	margin, err := reader.GetCampaignMargin(ctx, campaignID)
	if err != nil || !margin.MarginBreach {
		return nil
	}
	return []CampaignAdvisoryDTO{{
		Code:            "margin_floor_breach",
		Severity:        "warn",
		Message:         "RTB cost exceeds margin guard threshold for the last hour",
		RequiresConfirm: true,
		ImpactLabel:     campaign.FormatBasisPointsDisplay(margin.ThresholdBps),
		FloorPct:        float64(margin.ThresholdBps) / 100,
	}}
}

type PlacementBlockSuggestionDTO struct {
	PlacementID     string  `json:"placement_id"`
	Impressions     int64   `json:"impressions"`
	IVTRate         float64 `json:"ivt_rate"`
	IVTRateLabel    string  `json:"ivt_rate_label"`
	ReasonLabel     string  `json:"reason_label"`
	SuggestedAction string  `json:"suggested_action"`
}

type PlacementBlockSuggestionsResponseDTO struct {
	Items     []PlacementBlockSuggestionDTO `json:"items"`
	Freshness campaign.DataFreshnessDTO     `json:"freshness"`
}

type CampaignFlowValidateRequest struct {
	Paths json.RawMessage `json:"paths,omitempty"`
}

func postCampaignFlowValidate(h *campaign.CampaignsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req CampaignFlowValidateRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
			return
		}
	}
	var paths []campaign.FlowPathDTO
	if len(req.Paths) > 0 {
		paths, err = campaign.ParseFlowPaths(req.Paths)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid paths")
			return
		}
	} else {
		camp, err := h.Campaigns.GetCampaign(r.Context(), campaignID)
		if err != nil {
			h.WriteHandlerError(w, err)
			return
		}
		if camp.FlowID == "" || h.GetCampaignFlow == nil {
			httpresponse.JSON(w, http.StatusOK, campaign.FlowValidateResponseDTO{Valid: false, PathErrors: []campaign.FlowPathErrorDTO{{
				PathIndex: -1, Code: "missing_flow", Message: "campaign has no flow to validate",
			}}})
			return
		}
		flowID, err := uuid.Parse(camp.FlowID)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid flow id")
			return
		}
		flow, err := h.GetCampaignFlow(r.Context(), flowID)
		if err != nil {
			h.WriteHandlerError(w, err)
			return
		}
		paths, err = campaign.ParseFlowPaths(flow.Paths)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid stored flow paths")
			return
		}
	}
	resp := campaign.BuildCampaignFlowValidateResponse(paths)
	if h.ValidateCampaignFlowPaths != nil {
		if err := h.ValidateCampaignFlowPaths(r.Context(), paths); err != nil {
			resp.Valid = false
			resp.PathErrors = append(resp.PathErrors, flow.PathDBValidationErrors(err)...)
		}
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func postCampaignMacroPreview(h *campaign.CampaignsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[MacroPreviewRequestDTO](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	camp, err := h.Campaigns.GetCampaign(r.Context(), campaignID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	masked := campaign.MaskLevelFromContext(r.Context()) != authz.MaskFull
	preview, err := previewCampaignMacros(camp, req, masked)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, preview)
}

func getCampaignDiff(h *campaign.CampaignsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	against := strings.TrimSpace(r.URL.Query().Get("against"))
	if against == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "against query parameter required")
		return
	}
	otherID, err := uuid.Parse(against)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid against campaign id")
		return
	}
	if campaignID == otherID {
		httpresponse.JSON(w, http.StatusOK, CampaignDiffResponseDTO{Rows: nil})
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, otherID); err != nil {
			if errors.Is(err, campaign.ErrForbidden) || errors.Is(err, campaign.ErrCampaignNotFound) {
				httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "campaign not found")
				return
			}
			h.WriteHandlerError(w, err)
			return
		}
	}
	left, err := h.Campaigns.GetCampaign(r.Context(), campaignID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	right, err := h.Campaigns.GetCampaign(r.Context(), otherID)
	if err != nil {
		if errors.Is(err, campaign.ErrCampaignNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "campaign not found")
			return
		}
		h.WriteHandlerError(w, err)
		return
	}
	if left.CustomerID != "" && right.CustomerID != "" && left.CustomerID != right.CustomerID {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "campaign not found")
		return
	}
	masked := campaign.MaskLevelFromContext(r.Context()) != authz.MaskFull
	httpresponse.JSON(w, http.StatusOK, diffCampaignDTOs(left, right, masked))
}

func postCampaignClonePreview(h *campaign.CampaignsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[campaign.CloneCampaignHTTPRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	camp, err := h.Campaigns.GetCampaign(r.Context(), campaignID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, previewCloneCampaign(camp, req))
}

func postCampaignBulk(h *campaign.CampaignsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[BulkCampaignRequestDTO](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "pause" && action != "resume" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "unsupported bulk action")
		return
	}
	if len(req.CampaignIDs) == 0 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "campaign_ids required")
		return
	}
	if len(req.CampaignIDs) > bulkCampaignMaxSync {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "too many campaign_ids")
		return
	}
	results := make([]BulkCampaignResultRowDTO, 0, len(req.CampaignIDs))
	reason := "bulk_" + action
	for _, rawID := range req.CampaignIDs {
		row := BulkCampaignResultRowDTO{ID: rawID}
		campaignID, err := uuid.Parse(strings.TrimSpace(rawID))
		if err != nil {
			row.ErrorCode = "invalid_id"
			results = append(results, row)
			continue
		}
		if h.AuthorizeCampaignAccess != nil {
			if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
				row.ErrorCode = "forbidden"
				results = append(results, row)
				continue
			}
		}
		if action == "pause" {
			err = h.Campaigns.PauseCampaign(r.Context(), campaignID, reason)
		} else {
			err = h.Campaigns.ResumeCampaign(r.Context(), campaignID, reason)
		}
		if err != nil {
			row.ErrorCode = bulkCampaignErrorCode(err)
			results = append(results, row)
			continue
		}
		row.OK = true
		results = append(results, row)
	}
	httpresponse.JSON(w, http.StatusOK, BulkCampaignResponseDTO{Results: results})
}

func bulkCampaignErrorCode(err error) string {
	switch {
	case errors.Is(err, campaign.ErrCampaignNotFound):
		return "not_found"
	case errors.Is(err, campaign.ErrCampaignCannotBePaused), errors.Is(err, campaign.ErrCampaignNotPaused), errors.Is(err, campaign.ErrCampaignOutsideSchedule):
		return "invalid_state"
	case errors.Is(err, campaign.ErrCampaignPublishBlocked):
		return "publish_blocked"
	default:
		return "error"
	}
}

func getPlacementBlockSuggestions(h *campaign.CampaignsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	if h.ClickHouseQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := reports.ParseReportRange(r)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	suggestions, err := queryPlacementBlockSuggestions(r.Context(), h.ClickHouseQuery, campaignID, from, to, 20)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, PlacementBlockSuggestionsResponseDTO{
		Items:     suggestions,
		Freshness: reports.DataFreshnessFromClickHouse(r.Context(), h.ClickHouseQuery),
	})
}

const (
	placementBlockIVTThreshold   = 0.15
	placementBlockMinImpressions = int64(1000)
)

func queryPlacementBlockSuggestions(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignID uuid.UUID,
	from, to time.Time,
	limit int,
) ([]PlacementBlockSuggestionDTO, error) {
	if clickhouseQuery == nil {
		return nil, nil
	}
	campaignIDs := []uuid.UUID{campaignID}
	ivtRates, err := reports.QueryPlacementIVTRates(ctx, clickhouseQuery, campaignIDs, from, to)
	if err != nil {
		return nil, err
	}
	rows, _, err := reports.QueryPlacementReportRows(ctx, clickhouseQuery, campaignIDs, from, to, limit*4, 0)
	if err != nil {
		return nil, err
	}
	out := make([]PlacementBlockSuggestionDTO, 0, limit)
	for _, row := range rows {
		dto := reports.ToPlacementReportRowDTO(row, ivtRates[reports.ReportMetricsKey(row.Dimension, row.CampaignID)])
		if dto.Impressions < placementBlockMinImpressions || dto.IVTRate < placementBlockIVTThreshold {
			continue
		}
		out = append(out, PlacementBlockSuggestionDTO{
			PlacementID:     dto.PlacementID,
			Impressions:     dto.Impressions,
			IVTRate:         dto.IVTRate,
			IVTRateLabel:    campaign.FormatRateDisplay(dto.IVTRate),
			ReasonLabel:     "High IVT concentration",
			SuggestedAction: "block_placement",
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
