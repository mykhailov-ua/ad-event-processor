package reports

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedCampaignToggleFields = map[string]struct{}{
	"silent_reject_enabled":      {},
	"accept_lang_geo_enabled":    {},
	"json_serialization_enabled": {},
}

type CampaignToggleCohortKPIDTO struct {
	Window      string  `json:"window"`
	Impressions int64   `json:"impressions"`
	Rejects     int64   `json:"rejects"`
	Conversions int64   `json:"conversions"`
	ROIPct      float64 `json:"roi_pct"`
	DeltaLabel  string  `json:"delta_label,omitempty"`
	DeltaTone   string  `json:"delta_tone,omitempty"`
}

type CampaignToggleCohortReportResponse struct {
	CampaignID   string                       `json:"campaign_id"`
	ToggleField  string                       `json:"toggle_field"`
	ToggleAt     string                       `json:"toggle_at"`
	WindowHours  int                          `json:"window_hours"`
	Rows         []CampaignToggleCohortKPIDTO `json:"rows"`
	Freshness    DataFreshnessDTO             `json:"freshness"`
	Insufficient bool                         `json:"insufficient_data,omitempty"`
}

func (h *ReportsHTTPHandlers) registerCampaignToggleCohortReport(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/reports/campaign-toggle-cohort", limit(permAny([]string{"audit:read", "campaigns:read"}, h.wrapReport("campaign-toggle-cohort", h.getCampaignToggleCohortReport))))
}

func (h *ReportsHTTPHandlers) getCampaignToggleCohortReport(w http.ResponseWriter, r *http.Request) {
	campaignIDStr := strings.TrimSpace(r.URL.Query().Get("campaign_id"))
	if campaignIDStr == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "campaign_id is required")
		return
	}
	campaignID, err := uuid.Parse(campaignIDStr)
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
	toggleField := strings.TrimSpace(r.URL.Query().Get("toggle_field"))
	if toggleField == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "toggle_field is required")
		return
	}
	if _, ok := allowedCampaignToggleFields[toggleField]; !ok {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid toggle_field")
		return
	}
	if h.Pool == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "postgres not configured")
		return
	}
	toggleAt, auto, err := resolveCampaignToggleAt(r.Context(), h.Pool, campaignID, toggleField, strings.TrimSpace(r.URL.Query().Get("toggle_at")))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	windowHours := 72
	if raw := strings.TrimSpace(r.URL.Query().Get("window_hours")); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed <= 0 || parsed > 168 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid window_hours")
			return
		}
		windowHours = parsed
	}
	window := time.Duration(windowHours) * time.Hour
	beforeFrom := toggleAt.Add(-window)
	beforeTo := toggleAt
	afterFrom := toggleAt
	afterTo := toggleAt.Add(window)

	before, err := queryCampaignToggleWindowMetrics(r.Context(), h.Pool, h.ClickHouseQuery, campaignID, beforeFrom, beforeTo)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	after, err := queryCampaignToggleWindowMetrics(r.Context(), h.Pool, h.ClickHouseQuery, campaignID, afterFrom, afterTo)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	resp := CampaignToggleCohortReportResponse{
		CampaignID:  campaignID.String(),
		ToggleField: toggleField,
		ToggleAt:    toggleAt.UTC().Format(time.RFC3339),
		WindowHours: windowHours,
		Freshness:   h.reportFreshness(r.Context()),
	}
	if before.impressions == 0 {
		resp.Insufficient = true
		httpresponse.JSON(w, http.StatusOK, resp)
		return
	}
	beforeRow := CampaignToggleCohortKPIDTO{
		Window:      "before",
		Impressions: before.impressions,
		Rejects:     before.rejects,
		Conversions: before.conversions,
		ROIPct:      before.roiPct,
	}
	afterRow := CampaignToggleCohortKPIDTO{
		Window:      "after",
		Impressions: after.impressions,
		Rejects:     after.rejects,
		Conversions: after.conversions,
		ROIPct:      after.roiPct,
	}
	afterRow.DeltaLabel, afterRow.DeltaTone = formatToggleDelta(before.rejects, after.rejects)
	resp.Rows = []CampaignToggleCohortKPIDTO{beforeRow, afterRow}
	_ = auto
	httpresponse.JSON(w, http.StatusOK, resp)
}

type toggleWindowMetrics struct {
	impressions int64
	rejects     int64
	conversions int64
	roiPct      float64
}

func resolveCampaignToggleAt(ctx context.Context, pool *pgxpool.Pool, campaignID uuid.UUID, toggleField, toggleAtRaw string) (time.Time, bool, error) {
	if toggleAtRaw != "" {
		parsed, err := time.Parse(time.RFC3339, toggleAtRaw)
		if err != nil {
			return time.Time{}, false, invalidQueryError("invalid toggle_at")
		}
		return parsed.UTC(), false, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT created_at, changes
		FROM admin_audit_log
		WHERE action = 'UPDATE_CAMPAIGN_FRAUD'
		 AND target_id = $1
		ORDER BY created_at DESC
		LIMIT 8`,
		domain.ToUUID(campaignID),
	)
	if err != nil {
		return time.Time{}, true, err
	}
	defer rows.Close()
	var newerAt time.Time
	var newerSnap auditCampaignFraudChange
	first := true
	for rows.Next() {
		var createdAt time.Time
		var changes []byte
		if err := rows.Scan(&createdAt, &changes); err != nil {
			return time.Time{}, true, err
		}
		var snap auditCampaignFraudChange
		if err := json.Unmarshal(changes, &snap); err != nil {
			continue
		}
		if !first && toggleFieldChanged(toggleField, newerSnap, snap) {
			return newerAt, true, nil
		}
		newerSnap = snap
		newerAt = createdAt.UTC()
		first = false
	}
	return time.Time{}, true, invalidQueryError("toggle_at not found")
}

func toggleFieldChanged(field string, current, previous auditCampaignFraudChange) bool {
	switch field {
	case "silent_reject_enabled":
		return current.SilentRejectEnabled != previous.SilentRejectEnabled
	case "accept_lang_geo_enabled":
		return current.AcceptLangGeoEnabled != previous.AcceptLangGeoEnabled
	case "json_serialization_enabled":
		return current.JSONSerializationEnabled != previous.JSONSerializationEnabled
	default:
		return false
	}
}

func queryCampaignToggleWindowMetrics(
	ctx context.Context,
	pool *pgxpool.Pool,
	clickhouseQuery *database.ClickHouseQuery,
	campaignID uuid.UUID,
	from, to time.Time,
) (toggleWindowMetrics, error) {
	var out toggleWindowMetrics
	if pool == nil {
		return out, nil
	}
	err := pool.QueryRow(ctx, `
		SELECT
		 COALESCE(SUM(impressions_count), 0)::bigint,
		 COALESCE(SUM(conversions_count), 0)::bigint
		FROM campaign_stats
		WHERE campaign_id = $1
		 AND date >= $2::date
		 AND date < $3::date`,
		domain.ToUUID(campaignID),
		pgtype.Date{Time: from.UTC(), Valid: true},
		pgtype.Date{Time: to.UTC(), Valid: true},
	).Scan(&out.impressions, &out.conversions)
	if err != nil {
		return out, err
	}
	if clickhouseQuery != nil {
		rows, err := clickhouseQuery.Query(ctx, `
SELECT count() AS rejects
FROM fraud_events
WHERE campaign_id = ?
 AND created_at >= ?
 AND created_at < ?
 AND silent_reject_event = 0
 AND fraud_reason != ''`, campaignID, from, to)
		if err == nil {
			defer func() { _ = rows.Close() }()
			if rows.Next() {
				_ = rows.Scan(&out.rejects)
			}
		}
	}
	if out.impressions > 0 && out.conversions > 0 {
		out.roiPct = float64(out.conversions) / float64(out.impressions)
	}
	return out, nil
}

func formatToggleDelta(beforeRejects, afterRejects int64) (string, string) {
	if beforeRejects == 0 {
		return "n/a", "neutral"
	}
	delta := float64(afterRejects-beforeRejects) / float64(beforeRejects)
	label := formatRateDisplay(delta)
	if delta > 0.05 {
		return label, "negative"
	}
	if delta < -0.05 {
		return label, "positive"
	}
	return label, "neutral"
}
