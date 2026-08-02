package adminapi

import (
	"net/http"
	"time"
)

// ReportCompareDeltas holds period-over-period metric deltas (current − previous).
type ReportCompareDeltas struct {
	SpendMicroDelta   int64 `json:"spend_micro_delta"`
	RevenueMicroDelta int64 `json:"revenue_micro_delta"`
	ImpressionsDelta  int64 `json:"impressions_delta"`
	ClicksDelta       int64 `json:"clicks_delta"`
	ConversionsDelta  int64 `json:"conversions_delta"`
}

func parseComparePrevious(r *http.Request) bool {
	switch r.URL.Query().Get("compare") {
	case "previous", "1", "true":
		return true
	default:
		return r.URL.Query().Get("compare_period") == "true"
	}
}

func previousReportRange(from, to time.Time) (time.Time, time.Time) {
	span := to.Sub(from)
	if span <= 0 {
		span = defaultReportLookback
	}
	prevTo := from
	prevFrom := from.Add(-span)
	return prevFrom, prevTo
}

func placementRowKey(placementID, campaignID string) string {
	return placementID + "\x00" + campaignID
}

func keywordRowKey(keyword, campaignID string) string {
	return keyword + "\x00" + campaignID
}

func attachPlacementCompareDeltas(rows []PlacementReportRowDTO, prev []placementReportCHRow) {
	if len(prev) == 0 {
		return
	}
	prevByKey := make(map[string]placementReportCHRow, len(prev))
	for _, row := range prev {
		prevByKey[placementRowKey(row.PlacementID, row.CampaignID)] = row
	}
	for i := range rows {
		prevRow, ok := prevByKey[placementRowKey(rows[i].PlacementID, rows[i].CampaignID)]
		if !ok {
			continue
		}
		rows[i].Compare = &ReportCompareDeltas{
			SpendMicroDelta:   rows[i].SpendMicro - prevRow.SpendMicro,
			RevenueMicroDelta: rows[i].RevenueMicro - prevRow.RevenueMicro,
			ImpressionsDelta:  rows[i].Impressions - prevRow.Impressions,
			ClicksDelta:       rows[i].Clicks - prevRow.Clicks,
			ConversionsDelta:  rows[i].Conversions - prevRow.Conversions,
		}
	}
}

func attachKeywordCompareDeltas(rows []KeywordReportRowDTO, prev []keywordReportCHRow) {
	if len(prev) == 0 {
		return
	}
	prevByKey := make(map[string]keywordReportCHRow, len(prev))
	for _, row := range prev {
		prevByKey[keywordRowKey(row.Keyword, row.CampaignID)] = row
	}
	for i := range rows {
		prevRow, ok := prevByKey[keywordRowKey(rows[i].Keyword, rows[i].CampaignID)]
		if !ok {
			continue
		}
		rows[i].Compare = &ReportCompareDeltas{
			SpendMicroDelta:   rows[i].SpendMicro - prevRow.SpendMicro,
			RevenueMicroDelta: rows[i].RevenueMicro - prevRow.RevenueMicro,
			ImpressionsDelta:  rows[i].Impressions - prevRow.Impressions,
			ClicksDelta:       rows[i].Clicks - prevRow.Clicks,
			ConversionsDelta:  rows[i].Conversions - prevRow.Conversions,
		}
	}
}
