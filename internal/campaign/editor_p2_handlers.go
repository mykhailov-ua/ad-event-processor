package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/reports"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type CampaignFlowPathValidator func(ctx context.Context, paths []FlowPathDTO) error

type CampaignFlowValidateRequest struct {
	Paths json.RawMessage `json:"paths,omitempty"`
}

func (h *CampaignsHTTPHandlers) postCampaignFlowValidate(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
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
	var paths []FlowPathDTO
	if len(req.Paths) > 0 {
		paths, err = ParseFlowPaths(req.Paths)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid paths")
			return
		}
	} else {
		campaign, err := h.Campaigns.GetCampaign(r.Context(), campaignID)
		if err != nil {
			h.writeServiceError(w, err)
			return
		}
		if campaign.FlowID == "" || h.GetCampaignFlow == nil {
			httpresponse.JSON(w, http.StatusOK, FlowValidateResponseDTO{Valid: false, PathErrors: []FlowPathErrorDTO{{
				PathIndex: -1, Code: "missing_flow", Message: "campaign has no flow to validate",
			}}})
			return
		}
		flowID, err := uuid.Parse(campaign.FlowID)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid flow id")
			return
		}
		flow, err := h.GetCampaignFlow(r.Context(), flowID)
		if err != nil {
			h.writeServiceError(w, err)
			return
		}
		paths, err = ParseFlowPaths(flow.Paths)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid stored flow paths")
			return
		}
	}
	resp := BuildCampaignFlowValidateResponse(paths)
	if h.ValidateCampaignFlowPaths != nil {
		if err := h.ValidateCampaignFlowPaths(r.Context(), paths); err != nil {
			resp.Valid = false
			resp.PathErrors = append(resp.PathErrors, flow.PathDBValidationErrors(err)...)
		}
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (h *CampaignsHTTPHandlers) postCampaignMacroPreview(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
	if !ok {
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[MacroPreviewRequestDTO](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	campaign, err := h.Campaigns.GetCampaign(r.Context(), campaignID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	masked := maskLevelFromContext(r.Context()) != authz.MaskFull
	preview, err := previewCampaignMacros(campaign, req, masked)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, preview)
}

func (h *CampaignsHTTPHandlers) getCampaignDiff(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
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
			if errors.Is(err, ErrForbidden) || errors.Is(err, ErrCampaignNotFound) {
				httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "campaign not found")
				return
			}
			h.writeServiceError(w, err)
			return
		}
	}
	left, err := h.Campaigns.GetCampaign(r.Context(), campaignID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	right, err := h.Campaigns.GetCampaign(r.Context(), otherID)
	if err != nil {
		if errors.Is(err, ErrCampaignNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "campaign not found")
			return
		}
		h.writeServiceError(w, err)
		return
	}
	if left.CustomerID != "" && right.CustomerID != "" && left.CustomerID != right.CustomerID {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "campaign not found")
		return
	}
	masked := maskLevelFromContext(r.Context()) != authz.MaskFull
	httpresponse.JSON(w, http.StatusOK, diffCampaignDTOs(left, right, masked))
}

func (h *CampaignsHTTPHandlers) postCampaignClonePreview(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
	if !ok {
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[CloneCampaignHTTPRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	campaign, err := h.Campaigns.GetCampaign(r.Context(), campaignID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, previewCloneCampaign(campaign, req))
}

func (h *CampaignsHTTPHandlers) postCampaignBulk(w http.ResponseWriter, r *http.Request) {
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
	case errors.Is(err, ErrCampaignNotFound):
		return "not_found"
	case errors.Is(err, ErrCampaignCannotBePaused), errors.Is(err, ErrCampaignNotPaused), errors.Is(err, ErrCampaignOutsideSchedule):
		return "invalid_state"
	case errors.Is(err, ErrCampaignPublishBlocked):
		return "publish_blocked"
	default:
		return "error"
	}
}

func (h *CampaignsHTTPHandlers) getPlacementBlockSuggestions(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
	if !ok {
		return
	}
	if h.ClickHouseQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := reports.ParseReportRange(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	suggestions, err := queryPlacementBlockSuggestions(r.Context(), h.ClickHouseQuery, campaignID, from, to, 20)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, PlacementBlockSuggestionsResponseDTO{
		Items:     suggestions,
		Freshness: reports.DataFreshnessFromClickHouse(r.Context(), h.ClickHouseQuery),
	})
}

const placementBlockIVTThreshold = 0.15
const placementBlockMinImpressions = int64(1000)

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
			IVTRateLabel:    formatRateDisplay(dto.IVTRate),
			ReasonLabel:     "High IVT concentration",
			SuggestedAction: "block_placement",
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func buildCampaignConflictResponse(current CampaignDTO, req PatchCampaignRequest) CampaignConflictResponseDTO {
	fields := conflictFieldsForPatch(current, req)
	return CampaignConflictResponseDTO{
		Error:          "campaign_revision_conflict",
		ServerRevision: campaignRevision(current.UpdatedAt),
		ConflictFields: fields,
		MergeHintLabel: "Reload the campaign and merge your changes",
		Current:        current,
	}
}

func conflictFieldsForPatch(current CampaignDTO, req PatchCampaignRequest) []string {
	var fields []string
	if req.Name != nil && *req.Name != current.Name {
		fields = append(fields, "name")
	}
	if req.Status != nil && !strings.EqualFold(*req.Status, current.Status) {
		fields = append(fields, "status")
	}
	if req.PacingMode != nil && !strings.EqualFold(*req.PacingMode, current.PacingMode) {
		fields = append(fields, "pacing_mode")
	}
	if req.BudgetLimit != nil && *req.BudgetLimit != current.BudgetLimit {
		fields = append(fields, "budget_limit")
	}
	if req.TargetURL != nil && *req.TargetURL != current.TargetURL {
		fields = append(fields, "target_url")
	}
	if len(fields) == 0 {
		fields = []string{"updated_at"}
	}
	return fields
}

func resolveExpectedRevision(r *http.Request, req *PatchCampaignRequest) {
	if req.ExpectedRevision != nil && strings.TrimSpace(*req.ExpectedRevision) != "" {
		return
	}
	ifMatch := strings.TrimSpace(r.Header.Get("If-Match"))
	if ifMatch != "" {
		req.ExpectedRevision = &ifMatch
	}
}
