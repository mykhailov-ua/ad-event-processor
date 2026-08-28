package campaign

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
)

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

func buildCampaignContextLinks(ctx context.Context, campaign CampaignDTO) []CampaignContextLinkDTO {
	links := []CampaignContextLinkDTO{
		{Href: fmt.Sprintf("/reports/campaign-overview?campaign_id=%s&customer_id=%s", campaign.ID, campaign.CustomerID), Label: "Campaign overview", ReportKey: "campaign-overview", DefaultRange: "7d", RequiredPermissions: []string{"campaigns:read"}},
		{Href: fmt.Sprintf("/reports/pacing-drift?campaign_id=%s&customer_id=%s", campaign.ID, campaign.CustomerID), Label: "Pacing drift", ReportKey: "pacing-drift", DefaultRange: "7d", RequiredPermissions: []string{"campaigns:read"}},
		{Href: fmt.Sprintf("/reports/wire-signal-breakdown?campaign_id=%s&customer_id=%s", campaign.ID, campaign.CustomerID), Label: "Wire signals", ReportKey: "wire-signal-breakdown", DefaultRange: "7d", RequiredPermissions: reports.ReportPermsFraudCustomer()},
	}
	if snap, ok := authz.SnapshotFromContext(ctx); ok && snap.Has("audit:read") && snap.Mask != authz.MaskMasked {
		links = append(links, CampaignContextLinkDTO{
			Href:  fmt.Sprintf("/reports/filter-rejects?campaign_id=%s&customer_id=%s", campaign.ID, campaign.CustomerID),
			Label: "Filter rejects", ReportKey: "filter-rejects", DefaultRange: "24h", RequiredPermissions: []string{"audit:read"},
		})
	}
	return links
}

func buildCampaignSchedulePreview(campaign CampaignDTO, now time.Time) CampaignSchedulePreviewDTO {
	tz := strings.TrimSpace(campaign.Timezone)
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
	if campaign.StartAt != "" {
		if parsed, err := time.Parse(time.RFC3339, campaign.StartAt); err == nil {
			startAt = &parsed
			if localNow.Before(parsed.In(loc)) {
				active = false
			}
		}
	}
	if campaign.EndAt != "" {
		if parsed, err := time.Parse(time.RFC3339, campaign.EndAt); err == nil {
			endAt = &parsed
			if !localNow.Before(parsed.In(loc)) {
				active = false
			}
		}
	}
	hours := len(campaign.DaypartHours)
	if hours > 0 {
		active = false
		for _, h := range campaign.DaypartHours {
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
		CurrentlyActive: active && strings.EqualFold(campaign.Status, "ACTIVE"),
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

func diffCampaignDTOs(left, right CampaignDTO, masked bool) CampaignDiffResponseDTO {
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

type CloneCampaignOptions struct {
	IncludeFlow            bool `json:"include_flow"`
	IncludePostbacks       bool `json:"include_postbacks"`
	IncludeFraud           bool `json:"include_fraud"`
	IncludePlacementBlocks bool `json:"include_placement_blocks"`
	ResetSpend             bool `json:"reset_spend"`
}

type CloneCampaignPreviewDTO struct {
	SourceID    string               `json:"source_id"`
	Name        string               `json:"name"`
	WouldCreate CloneCampaignOptions `json:"would_create"`
}

func previewCloneCampaign(campaign CampaignDTO, req CloneCampaignHTTPRequest) CloneCampaignPreviewDTO {
	opts := normalizedCloneOptions(req.Options)
	name := cloneCampaignName(campaign.Name, req.NamePrefix, req.NameSuffix)
	return CloneCampaignPreviewDTO{
		SourceID:    campaign.ID,
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

func marginAdvisoryForCampaign(ctx context.Context, campaignID uuid.UUID, reader CampaignReader) []CampaignAdvisoryDTO {
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
		ImpactLabel:     formatBasisPointsDisplay(margin.ThresholdBps),
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
	Freshness DataFreshnessDTO              `json:"freshness"`
}
