package reports

type CostSyncCoverageReportResponse struct {
	Rows       []CostCoverageRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO     `json:"freshness"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

type DiscrepancyBuySellRowDTO struct {
	CampaignID    string  `json:"campaign_id"`
	BuySpendMicro int64   `json:"buy_spend_micro"`
	SellRevMicro  int64   `json:"sell_rev_micro"`
	DeltaMicro    int64   `json:"delta_micro"`
	DeltaPct      float64 `json:"delta_pct"`
}

type DiscrepancyBuySellReportResponse struct {
	Rows       []DiscrepancyBuySellRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO           `json:"freshness"`
	NextCursor string                     `json:"next_cursor,omitempty"`
}

type CustomerPortfolioSummaryDTO struct {
	Active          int   `json:"active"`
	Paused          int   `json:"paused"`
	Archived        int   `json:"archived"`
	Impressions7d   int64 `json:"impressions_7d"`
	Clicks7d        int64 `json:"clicks_7d"`
	OverspendCount  int   `json:"overspend_count"`
	AttentionCount  int   `json:"attention_count"`
	CampaignsSample int   `json:"campaigns_sample"`
}

type CustomerPortfolioCampaignRowDTO struct {
	CampaignID     string  `json:"campaign_id"`
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	Impressions7d  int64   `json:"impressions_7d"`
	Clicks7d       int64   `json:"clicks_7d"`
	UtilizationPct float64 `json:"utilization_pct"`
	PacingDriftPct float64 `json:"pacing_drift_pct"`
	OverspendRisk  bool    `json:"overspend_risk"`
}

type CustomerPortfolioReportResponse struct {
	Summary   CustomerPortfolioSummaryDTO       `json:"summary"`
	Campaigns []CustomerPortfolioCampaignRowDTO `json:"campaigns"`
	Freshness DataFreshnessDTO                  `json:"freshness"`
}

func discrepancyBuySellRowsFromMaps(rows []map[string]any) []DiscrepancyBuySellRowDTO {
	out := make([]DiscrepancyBuySellRowDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, DiscrepancyBuySellRowDTO{
			CampaignID:    mapRowString(row, "campaign_id"),
			BuySpendMicro: mapRowInt64(row, "buy_spend_micro"),
			SellRevMicro:  mapRowInt64(row, "sell_rev_micro"),
			DeltaMicro:    mapRowInt64(row, "delta_micro"),
			DeltaPct:      mapRowFloat64(row, "delta_pct"),
		})
	}
	return out
}

func customerPortfolioSummaryFromDTO(portfolio BuyerPortfolioDTO) CustomerPortfolioSummaryDTO {
	return CustomerPortfolioSummaryDTO{
		Active:          portfolio.Active,
		Paused:          portfolio.Paused,
		Archived:        portfolio.Archived,
		Impressions7d:   portfolio.Impressions7d,
		Clicks7d:        portfolio.Clicks7d,
		OverspendCount:  portfolio.OverspendCount,
		AttentionCount:  len(portfolio.Attention),
		CampaignsSample: len(portfolio.Campaigns),
	}
}

func customerPortfolioCampaignRowsFromDTO(campaigns []BuyerCampaignPortfolioRowDTO) []CustomerPortfolioCampaignRowDTO {
	out := make([]CustomerPortfolioCampaignRowDTO, 0, len(campaigns))
	for _, c := range campaigns {
		out = append(out, CustomerPortfolioCampaignRowDTO{
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
