package reports

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

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

func ReportMetricsKey(dimension, campaignID string) string {
	return dimension + "\x00" + campaignID
}

func compareReportMetrics(cur, prev reportMetricsCHRow) ReportCompareDeltas {
	return ReportCompareDeltas{
		SpendMicroDelta:   cur.SpendMicro - prev.SpendMicro,
		RevenueMicroDelta: cur.RevenueMicro - prev.RevenueMicro,
		ImpressionsDelta:  cur.Impressions - prev.Impressions,
		ClicksDelta:       cur.Clicks - prev.Clicks,
		ConversionsDelta:  cur.Conversions - prev.Conversions,
	}
}

func attachReportCompareDeltas[Row any](
	rows []Row,
	prev []reportMetricsCHRow,
	rowKey func(Row) string,
	currentMetrics func(Row) reportMetricsCHRow,
	setCompare func(*Row, ReportCompareDeltas),
) {
	if len(prev) == 0 {
		return
	}
	prevByKey := make(map[string]reportMetricsCHRow, len(prev))
	for _, row := range prev {
		prevByKey[ReportMetricsKey(row.Dimension, row.CampaignID)] = row
	}
	for i := range rows {
		prevRow, ok := prevByKey[rowKey(rows[i])]
		if !ok {
			continue
		}
		d := compareReportMetrics(currentMetrics(rows[i]), prevRow)
		setCompare(&rows[i], d)
	}
}

func reportMetricsFromPlacementDTO(row PlacementReportRowDTO) reportMetricsCHRow {
	return reportMetricsCHRow{
		Dimension:    row.PlacementID,
		CampaignID:   row.CampaignID,
		Impressions:  row.Impressions,
		Clicks:       row.Clicks,
		Conversions:  row.Conversions,
		SpendMicro:   row.SpendMicro,
		RevenueMicro: row.RevenueMicro,
	}
}

func reportMetricsFromKeywordDTO(row KeywordReportRowDTO) reportMetricsCHRow {
	return reportMetricsCHRow{
		Dimension:    row.Keyword,
		CampaignID:   row.CampaignID,
		Impressions:  row.Impressions,
		Clicks:       row.Clicks,
		Conversions:  row.Conversions,
		SpendMicro:   row.SpendMicro,
		RevenueMicro: row.RevenueMicro,
	}
}

func attachPlacementCompareDeltas(rows []PlacementReportRowDTO, prev []reportMetricsCHRow) {
	attachReportCompareDeltas(rows, prev,
		func(r PlacementReportRowDTO) string { return ReportMetricsKey(r.PlacementID, r.CampaignID) },
		reportMetricsFromPlacementDTO,
		func(r *PlacementReportRowDTO, d ReportCompareDeltas) { r.Compare = &d },
	)
}

func attachKeywordCompareDeltas(rows []KeywordReportRowDTO, prev []reportMetricsCHRow) {
	attachReportCompareDeltas(rows, prev,
		func(r KeywordReportRowDTO) string { return ReportMetricsKey(r.Keyword, r.CampaignID) },
		reportMetricsFromKeywordDTO,
		func(r *KeywordReportRowDTO, d ReportCompareDeltas) { r.Compare = &d },
	)
}

func reportMetricsFromTrafficDTO(row TrafficSourceRowDTO) reportMetricsCHRow {
	return reportMetricsCHRow{
		Dimension:    row.Channel,
		Impressions:  row.Impressions,
		Clicks:       row.Clicks,
		Conversions:  row.Conversions,
		SpendMicro:   row.SpendMicro,
		RevenueMicro: row.RevenueMicro,
	}
}

func attachTrafficCompareDeltas(rows []TrafficSourceRowDTO, prev []TrafficSourceRowDTO) {
	if len(prev) == 0 {
		return
	}
	prevByChannel := make(map[string]TrafficSourceRowDTO, len(prev))
	for _, row := range prev {
		prevByChannel[row.Channel] = row
	}
	for i := range rows {
		prevRow, ok := prevByChannel[rows[i].Channel]
		if !ok {
			continue
		}
		d := compareReportMetrics(reportMetricsFromTrafficDTO(rows[i]), reportMetricsFromTrafficDTO(prevRow))
		rows[i].Compare = &d
	}
}

func reportMetricsFromGeoDTO(row GeoROIRowDTO) reportMetricsCHRow {
	return reportMetricsCHRow{
		Dimension:    row.Country,
		Impressions:  row.Impressions,
		Clicks:       row.Clicks,
		Conversions:  row.Conversions,
		SpendMicro:   row.SpendMicro,
		RevenueMicro: row.RevenueMicro,
	}
}

func attachGeoCompareDeltas(rows []GeoROIRowDTO, prev []GeoROIRowDTO) {
	if len(prev) == 0 {
		return
	}
	prevByCountry := make(map[string]GeoROIRowDTO, len(prev))
	for _, row := range prev {
		prevByCountry[row.Country] = row
	}
	for i := range rows {
		prevRow, ok := prevByCountry[rows[i].Country]
		if !ok {
			continue
		}
		d := compareReportMetrics(reportMetricsFromGeoDTO(rows[i]), reportMetricsFromGeoDTO(prevRow))
		rows[i].Compare = &d
	}
}

func compareDeltasToMap(d ReportCompareDeltas) map[string]any {
	return map[string]any{
		"spend_micro_delta":   d.SpendMicroDelta,
		"revenue_micro_delta": d.RevenueMicroDelta,
		"impressions_delta":   d.ImpressionsDelta,
		"clicks_delta":        d.ClicksDelta,
		"conversions_delta":   d.ConversionsDelta,
	}
}

func mapRowMetrics(row map[string]any) reportMetricsCHRow {
	return reportMetricsCHRow{
		Dimension:    mapRowString(row, "placement_id", "channel", "country", "hour", "campaign_id"),
		CampaignID:   mapRowString(row, "campaign_id"),
		Impressions:  mapRowInt64(row, "impressions"),
		Clicks:       mapRowInt64(row, "clicks"),
		Conversions:  mapRowInt64(row, "conversions"),
		SpendMicro:   mapRowInt64(row, "spend_micro", "ad_spend_micro"),
		RevenueMicro: mapRowInt64(row, "revenue_micro"),
	}
}

func mapRowString(row map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := row[key]; ok && v != nil {
			switch t := v.(type) {
			case string:
				if t != "" {
					return t
				}
			default:
				s := fmt.Sprint(t)
				if s != "" && s != "<nil>" {
					return s
				}
			}
		}
	}
	return ""
}

func mapRowInt64(row map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if v, ok := row[key]; ok && v != nil {
			switch t := v.(type) {
			case int64:
				return t
			case int:
				return int64(t)
			case float64:
				return int64(t)
			case uint64:
				return int64(t)
			}
		}
	}
	return 0
}

func mapReportRowKey(row map[string]any, keyFields ...string) string {
	if len(keyFields) == 0 {
		return mapRowMetrics(row).Dimension
	}
	var b strings.Builder
	for i, field := range keyFields {
		if i > 0 {
			b.WriteByte('\x00')
		}
		b.WriteString(mapRowString(row, field))
	}
	return b.String()
}

func attachMapCompareDeltas(rows, prev []map[string]any, keyFields ...string) {
	if len(prev) == 0 {
		return
	}
	prevByKey := make(map[string]reportMetricsCHRow, len(prev))
	for _, row := range prev {
		prevByKey[mapReportRowKey(row, keyFields...)] = mapRowMetrics(row)
	}
	for i := range rows {
		prevRow, ok := prevByKey[mapReportRowKey(rows[i], keyFields...)]
		if !ok {
			continue
		}
		d := compareReportMetrics(mapRowMetrics(rows[i]), prevRow)
		rows[i]["compare"] = compareDeltasToMap(d)
	}
}
