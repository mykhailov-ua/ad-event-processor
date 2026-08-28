package campaign

import (
	"fmt"
	"strconv"
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
