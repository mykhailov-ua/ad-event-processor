package campaign

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

type CampaignFraudEditorCardDTO struct {
	SignalKey  string `json:"signal_key"`
	TitleLabel string `json:"title_label"`
	CountLabel string `json:"count_label"`
	Severity   string `json:"severity"`
	ReportKey  string `json:"report_key,omitempty"`
	ReportHref string `json:"report_href,omitempty"`
}

type CampaignFraudEditorSummaryDTO struct {
	CampaignID string                       `json:"campaign_id"`
	Cards      []CampaignFraudEditorCardDTO `json:"cards"`
	Disclaimer string                       `json:"disclaimer,omitempty"`
}

func BuildCampaignFraudEditorSummary(camp CampaignDTO, preview CampaignFraudPreviewDTO) CampaignFraudEditorSummaryDTO {
	return buildCampaignFraudEditorSummary(camp, preview)
}

func buildCampaignFraudEditorSummary(campaign CampaignDTO, preview CampaignFraudPreviewDTO) CampaignFraudEditorSummaryDTO {
	cards := make([]CampaignFraudEditorCardDTO, 0, 3)
	if preview.ByTier.Suspect > 0 {
		cards = append(cards, CampaignFraudEditorCardDTO{
			SignalKey:  "suspect_tier",
			TitleLabel: "Suspect tier",
			CountLabel: fraudEditorCountLabel(preview.ByTier.Suspect),
			Severity:   "warn",
			ReportKey:  "signal-effectiveness",
			ReportHref: fraudEditorReportHref(campaign, "signal-effectiveness"),
		})
	}
	if preview.ByTier.IVT > 0 {
		cards = append(cards, CampaignFraudEditorCardDTO{
			SignalKey:  "ivt_tier",
			TitleLabel: "IVT tier",
			CountLabel: fraudEditorCountLabel(preview.ByTier.IVT),
			Severity:   "fail",
			ReportKey:  "wire-signal-breakdown",
			ReportHref: fraudEditorReportHref(campaign, "wire-signal-breakdown"),
		})
	}
	if preview.ByTier.Block > 0 {
		cards = append(cards, CampaignFraudEditorCardDTO{
			SignalKey:  "block_tier",
			TitleLabel: "Block tier",
			CountLabel: fraudEditorCountLabel(preview.ByTier.Block),
			Severity:   "fail",
			ReportKey:  "customer-fraud-by-type",
			ReportHref: fraudEditorReportHref(campaign, "customer-fraud-by-type"),
		})
	}
	return CampaignFraudEditorSummaryDTO{
		CampaignID: campaign.ID,
		Cards:      cards,
		Disclaimer: preview.Disclaimer,
	}
}

func fraudEditorCountLabel(n int64) string {
	if n == 1 {
		return "1 IP"
	}
	return strconv.FormatInt(n, 10) + " IPs"
}

func fraudEditorReportHref(campaign CampaignDTO, reportKey string) string {
	return fmt.Sprintf("/reports/%s?campaign_id=%s&customer_id=%s", reportKey, campaign.ID, campaign.CustomerID)
}

func (h *CampaignsHTTPHandlers) registerCampaignFraudRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.CampaignFraud == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/campaigns/{id}/fraud", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, h.getCampaignFraud)))
	mux.HandleFunc("PATCH /api/v1/campaigns/{id}/fraud", limit(perm([]string{"campaigns:write"}, h.patchCampaignFraud)))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/fraud/preview", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, h.postCampaignFraudPreview)))
}

func (h *CampaignsHTTPHandlers) getCampaignFraud(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	cfg, err := h.CampaignFraud.GetCampaignFraudConfig(r.Context(), campaignID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, cfg)
}

func (h *CampaignsHTTPHandlers) patchCampaignFraud(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := decodePatchCampaignFraudRequest(body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Preset != nil {
		preset := strings.TrimSpace(strings.ToLower(*req.Preset))
		if preset == "" {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "preset must not be empty")
			return
		}
		req.Preset = &preset
	}
	updated, err := h.CampaignFraud.UpdateCampaignFraudConfig(r.Context(), campaignID, req)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, updated)
}

func (h *CampaignsHTTPHandlers) postCampaignFraudPreview(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	if h.AllowFraudPreview != nil && !h.AllowFraudPreview(campaignID.String()) {
		httpresponse.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "fraud preview rate limit exceeded")
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[PreviewCampaignFraudRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Preset != nil {
		preset := strings.TrimSpace(strings.ToLower(*req.Preset))
		if preset == "" {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "preset must not be empty")
			return
		}
		req.Preset = &preset
	}
	out, err := h.CampaignFraud.PreviewCampaignFraudImpact(r.Context(), campaignID, req)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, out)
}
