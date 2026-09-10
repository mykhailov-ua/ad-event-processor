package reports

type SpendVelocityRowDTO struct {
	Bucket     string               `json:"bucket"`
	SpendMicro int64                `json:"spend_micro"`
	Clicks     int64                `json:"clicks"`
	Compare    *ReportCompareDeltas `json:"compare,omitempty"`
}

type SpendVelocityReportResponse struct {
	Rows       []SpendVelocityRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO      `json:"freshness"`
	NextCursor string                `json:"next_cursor,omitempty"`
}

type DaypartHeatmapRowDTO struct {
	Hour    int                  `json:"hour"`
	Clicks  int64                `json:"clicks"`
	Compare *ReportCompareDeltas `json:"compare,omitempty"`
}

type DaypartHeatmapReportResponse struct {
	Rows       []DaypartHeatmapRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO       `json:"freshness"`
	NextCursor string                 `json:"next_cursor,omitempty"`
}

type CampaignGeoDeviceRowDTO struct {
	Country string `json:"country"`
	Device  string `json:"device"`
	Clicks  int64  `json:"clicks"`
}

type CampaignGeoDeviceReportResponse struct {
	Rows       []CampaignGeoDeviceRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO          `json:"freshness"`
	NextCursor string                    `json:"next_cursor,omitempty"`
}

type TrueROIReportRowDTO struct {
	CampaignID      string               `json:"campaign_id"`
	AdSpendMicro    int64                `json:"ad_spend_micro"`
	RevenueMicro    int64                `json:"revenue_micro"`
	TrueProfitMicro int64                `json:"true_profit_micro"`
	TrueRoiPct      float64              `json:"true_roi_pct"`
	TrueCpaMicro    int64                `json:"true_cpa_micro"`
	Conversions     int64                `json:"conversions"`
	Compare         *ReportCompareDeltas `json:"compare,omitempty"`
}

type TrueROIReportResponse struct {
	Rows       []TrueROIReportRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO      `json:"freshness"`
	NextCursor string                `json:"next_cursor,omitempty"`
}

type CampaignOverviewRowDTO struct {
	CampaignID     string  `json:"campaign_id"`
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	Impressions7d  int64   `json:"impressions_7d"`
	Clicks7d       int64   `json:"clicks_7d"`
	UtilizationPct float64 `json:"utilization_pct"`
	PacingDriftPct float64 `json:"pacing_drift_pct"`
	OverspendRisk  bool    `json:"overspend_risk"`
}

type CampaignOverviewReportResponse struct {
	Rows      []CampaignOverviewRowDTO `json:"rows"`
	Freshness DataFreshnessDTO         `json:"freshness"`
}

func spendVelocityRowsFromMaps(rows []map[string]any) []SpendVelocityRowDTO {
	out := make([]SpendVelocityRowDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, SpendVelocityRowDTO{
			Bucket:     mapRowString(row, "bucket"),
			SpendMicro: mapRowInt64(row, "spend_micro"),
			Clicks:     mapRowInt64(row, "clicks"),
		})
	}
	return out
}

func daypartHeatmapRowsFromMaps(rows []map[string]any) []DaypartHeatmapRowDTO {
	out := make([]DaypartHeatmapRowDTO, 0, len(rows))
	for _, row := range rows {
		hour := int(mapRowInt64(row, "hour"))
		out = append(out, DaypartHeatmapRowDTO{
			Hour:   hour,
			Clicks: mapRowInt64(row, "clicks"),
		})
	}
	return out
}

func campaignGeoDeviceRowsFromMaps(rows []map[string]any) []CampaignGeoDeviceRowDTO {
	out := make([]CampaignGeoDeviceRowDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, CampaignGeoDeviceRowDTO{
			Country: mapRowString(row, "country"),
			Device:  mapRowString(row, "device"),
			Clicks:  mapRowInt64(row, "clicks"),
		})
	}
	return out
}

func redactTrueROIRows(rows []TrueROIReportRowDTO) []TrueROIReportRowDTO {
	for i := range rows {
		rows[i].RevenueMicro = 0
		rows[i].TrueProfitMicro = 0
		rows[i].TrueRoiPct = 0
		rows[i].TrueCpaMicro = 0
	}
	return rows
}

func trueROIRowsFromMaps(rows []map[string]any) []TrueROIReportRowDTO {
	out := make([]TrueROIReportRowDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, TrueROIReportRowDTO{
			CampaignID:      mapRowString(row, "campaign_id"),
			AdSpendMicro:    mapRowInt64(row, "ad_spend_micro"),
			RevenueMicro:    mapRowInt64(row, "revenue_micro"),
			TrueProfitMicro: mapRowInt64(row, "true_profit_micro"),
			TrueRoiPct:      mapRowFloat64(row, "true_roi_pct"),
			TrueCpaMicro:    mapRowInt64(row, "true_cpa_micro"),
			Conversions:     mapRowInt64(row, "conversions"),
		})
	}
	return out
}

func campaignOverviewRowsFromPortfolio(campaigns []BuyerCampaignPortfolioRowDTO) []CampaignOverviewRowDTO {
	out := make([]CampaignOverviewRowDTO, 0, len(campaigns))
	for _, c := range campaigns {
		out = append(out, CampaignOverviewRowDTO{
			CampaignID:     c.ID,
			Name:           c.Name,
			Status:         c.Status,
			Impressions7d:  c.Impressions7d,
			Clicks7d:       c.Clicks7d,
			UtilizationPct: c.UtilizationPct,
			PacingDriftPct: c.PacingDriftPct,
			OverspendRisk:  c.OverspendRisk,
		})
	}
	return out
}

func mapRowFloat64(row map[string]any, keys ...string) float64 {
	for _, key := range keys {
		if v, ok := row[key]; ok && v != nil {
			switch t := v.(type) {
			case float64:
				return t
			case float32:
				return float64(t)
			case int64:
				return float64(t)
			case int:
				return float64(t)
			}
		}
	}
	return 0
}

func attachSpendVelocityCompareDeltas(rows []SpendVelocityRowDTO, prev []SpendVelocityRowDTO) {
	if len(prev) == 0 {
		return
	}
	prevByBucket := make(map[string]SpendVelocityRowDTO, len(prev))
	for _, row := range prev {
		prevByBucket[row.Bucket] = row
	}
	for i := range rows {
		prevRow, ok := prevByBucket[rows[i].Bucket]
		if !ok {
			continue
		}
		d := ReportCompareDeltas{
			SpendMicroDelta: rows[i].SpendMicro - prevRow.SpendMicro,
			ClicksDelta:     rows[i].Clicks - prevRow.Clicks,
		}
		rows[i].Compare = &d
	}
}

func attachDaypartHeatmapCompareDeltas(rows []DaypartHeatmapRowDTO, prev []DaypartHeatmapRowDTO) {
	if len(prev) == 0 {
		return
	}
	prevByHour := make(map[int]DaypartHeatmapRowDTO, len(prev))
	for _, row := range prev {
		prevByHour[row.Hour] = row
	}
	for i := range rows {
		prevRow, ok := prevByHour[rows[i].Hour]
		if !ok {
			continue
		}
		d := ReportCompareDeltas{
			ClicksDelta: rows[i].Clicks - prevRow.Clicks,
		}
		rows[i].Compare = &d
	}
}

func attachTrueROICompareDeltas(rows []TrueROIReportRowDTO, prev []TrueROIReportRowDTO) {
	if len(prev) == 0 {
		return
	}
	prevByCampaign := make(map[string]TrueROIReportRowDTO, len(prev))
	for _, row := range prev {
		prevByCampaign[row.CampaignID] = row
	}
	for i := range rows {
		prevRow, ok := prevByCampaign[rows[i].CampaignID]
		if !ok {
			continue
		}
		d := ReportCompareDeltas{
			SpendMicroDelta:   rows[i].AdSpendMicro - prevRow.AdSpendMicro,
			RevenueMicroDelta: rows[i].RevenueMicro - prevRow.RevenueMicro,
			ConversionsDelta:  rows[i].Conversions - prevRow.Conversions,
		}
		rows[i].Compare = &d
	}
}
