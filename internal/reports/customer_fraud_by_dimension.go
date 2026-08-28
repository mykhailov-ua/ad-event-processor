package reports

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

const maxFraudDimensionRows = 500

var allowedFraudDimensions = map[string]struct{}{
	"placement": {},
	"sub1":      {},
	"sub2":      {},
	"country":   {},
	"campaign":  {},
}

type CustomerFraudByDimensionRowDTO struct {
	DimensionValue        string  `json:"dimension_value"`
	CampaignID            string  `json:"campaign_id,omitempty"`
	Impressions           int64   `json:"impressions"`
	Clicks                int64   `json:"clicks"`
	IVTEvents             int64   `json:"ivt_events"`
	BlockedEvents         int64   `json:"blocked_events"`
	IVTRate               float64 `json:"ivt_rate"`
	IVTRateLabel          string  `json:"ivt_rate_label"`
	TopFraudCategory      string  `json:"top_fraud_category,omitempty"`
	TopFraudCategoryLabel string  `json:"top_fraud_category_label,omitempty"`
	DeltaLabel            string  `json:"delta_label,omitempty"`
	DeltaTone             string  `json:"delta_tone,omitempty"`
}

type CustomerFraudByDimensionReportResponse struct {
	Rows       []CustomerFraudByDimensionRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO                 `json:"freshness"`
	Truncated  bool                             `json:"truncated,omitempty"`
	NextCursor string                           `json:"next_cursor,omitempty"`
}

func (h *ReportsHTTPHandlers) registerCustomerFraudByDimensionReport(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/reports/customer-fraud-by-dimension", limit(permAny(ReportPermsFraudCustomer(), h.wrapReport("customer-fraud-by-dimension", h.getCustomerFraudByDimensionReport))))
}

func (h *ReportsHTTPHandlers) getCustomerFraudByDimensionReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.resolveReportCustomerID(w, r)
	if !ok {
		return
	}
	dimension := strings.TrimSpace(r.URL.Query().Get("dimension"))
	if dimension == "" {
		dimension = "placement"
	}
	if _, ok := allowedFraudDimensions[dimension]; !ok {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid dimension")
		return
	}
	if h.ClickHouseQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := ParseReportRange(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	campaignFilter := strings.TrimSpace(r.URL.Query().Get("campaign_id"))
	campaignIDs, err := listCustomerCampaignIDs(r.Context(), h.Pool, customerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if campaignFilter != "" {
		filterID, parseErr := uuid.Parse(campaignFilter)
		if parseErr != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
			return
		}
		allowed := false
		for _, id := range campaignIDs {
			if id == filterID {
				allowed = true
				break
			}
		}
		if !allowed {
			h.writeServiceError(w, ErrForbidden)
			return
		}
		campaignIDs = []uuid.UUID{filterID}
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, CustomerFraudByDimensionReportResponse{
			Rows:      []CustomerFraudByDimensionRowDTO{},
			Freshness: h.reportFreshness(r.Context()),
		})
		return
	}
	compare := strings.TrimSpace(r.URL.Query().Get("compare")) == "1"
	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()
	rows, truncated, err := buildCustomerFraudByDimensionRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, dimension, r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if compare {
		window := to.Sub(from)
		priorFrom := from.Add(-window)
		priorRows, _, priorErr := buildCustomerFraudByDimensionRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, priorFrom, from, dimension, r.Context())
		if priorErr == nil {
			applyDimensionCompareDeltas(rows, priorRows)
		}
	}
	page, err := coldpath.ParseCursorPagination(r, 50, maxFraudDimensionRows)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	total := int64(len(rows))
	if page.Offset >= len(rows) {
		rows = nil
	} else {
		end := page.Offset + page.Limit
		if end > len(rows) {
			end = len(rows)
		}
		rows = rows[page.Offset:end]
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, CustomerFraudByDimensionReportResponse{
		Rows:       rows,
		Freshness:  h.reportFreshness(r.Context()),
		Truncated:  truncated,
		NextCursor: nextCursor,
	})
}

type dimensionAggKey struct {
	campaignID string
	value      string
}

func buildCustomerFraudByDimensionRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	dimension string,
	scrubCtx context.Context,
) ([]CustomerFraudByDimensionRowDTO, bool, error) {
	topCategoryByCampaign := make(map[string]dimensionTopCategorySlot)
	switch dimension {
	case "placement", "campaign":
		rawRows, _, err := queryFraudBreakdownRows(ctx, clickhouseQuery, campaignIDs, from, to, 10_000, 0)
		if err != nil {
			return nil, false, err
		}
		agg := make(map[dimensionAggKey]*CustomerFraudByDimensionRowDTO)
		for _, row := range rawRows {
			value := row.CampaignID
			if dimension == "placement" {
				value = row.PlacementID
				if value == "" {
					continue
				}
			}
			key := dimensionAggKey{campaignID: row.CampaignID, value: value}
			slot := agg[key]
			if slot == nil {
				slot = &CustomerFraudByDimensionRowDTO{
					DimensionValue: value,
					CampaignID:     row.CampaignID,
				}
				agg[key] = slot
			}
			slot.BlockedEvents += row.EventCount
			slot.IVTEvents += row.SilentRejectCount
			category, label := FraudReasonToCategory(row.FraudReason)
			if slot.TopFraudCategory == "" || row.EventCount > 0 {
				slot.TopFraudCategory = category
				slot.TopFraudCategoryLabel = label
			}
			if row.EventCount > topCategoryByCampaign[row.CampaignID].events {
				topCategoryByCampaign[row.CampaignID] = dimensionTopCategorySlot{
					category: category,
					events:   row.EventCount,
				}
			}
		}
		return dimensionRowsFromAgg(agg), len(agg) > maxFraudDimensionRows, nil
	default:
		ivtRows, _, err := queryIVTBySourceRows(ctx, clickhouseQuery, campaignIDs, from, to, maxFraudDimensionRows, 0)
		if err != nil {
			return nil, false, err
		}
		rawRows, _, err := queryFraudBreakdownRows(ctx, clickhouseQuery, campaignIDs, from, to, 5_000, 0)
		if err != nil {
			return nil, false, err
		}
		for _, row := range rawRows {
			category, _ := FraudReasonToCategory(row.FraudReason)
			prev := topCategoryByCampaign[row.CampaignID]
			if row.EventCount > prev.events {
				topCategoryByCampaign[row.CampaignID] = dimensionTopCategorySlot{
					category: category,
					events:   row.EventCount,
				}
			}
		}
		agg := make(map[dimensionAggKey]*CustomerFraudByDimensionRowDTO)
		for _, row := range ivtRows {
			value := dimensionValueFromIVT(row, dimension)
			if value == "" {
				continue
			}
			key := dimensionAggKey{campaignID: row.CampaignID, value: value}
			slot := agg[key]
			if slot == nil {
				slot = &CustomerFraudByDimensionRowDTO{
					DimensionValue: value,
					CampaignID:     row.CampaignID,
				}
				if top, ok := topCategoryByCampaign[row.CampaignID]; ok {
					slot.TopFraudCategory = top.category
					slot.TopFraudCategoryLabel = FraudCategoryLabel(top.category)
				}
				agg[key] = slot
			}
			slot.Impressions += row.Impressions
			slot.Clicks += row.Clicks
			slot.IVTEvents += row.IVTEvents
		}
		return dimensionRowsFromAgg(agg), len(ivtRows) >= maxFraudDimensionRows, nil
	}
}

type dimensionTopCategorySlot struct {
	category string
	events   int64
}

func dimensionValueFromIVT(row ivtBySourceCHRow, dimension string) string {
	switch dimension {
	case "sub1":
		return row.Sub1
	case "sub2":
		return row.Sub2
	case "country":
		return row.Country
	default:
		return ""
	}
}

func dimensionRowsFromAgg(agg map[dimensionAggKey]*CustomerFraudByDimensionRowDTO) []CustomerFraudByDimensionRowDTO {
	out := make([]CustomerFraudByDimensionRowDTO, 0, len(agg))
	for _, row := range agg {
		if row.Clicks > 0 {
			row.IVTRate = float64(row.IVTEvents) / float64(row.Clicks)
		}
		row.IVTRateLabel = formatRateDisplay(row.IVTRate)
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IVTEvents == out[j].IVTEvents {
			return out[i].DimensionValue < out[j].DimensionValue
		}
		return out[i].IVTEvents > out[j].IVTEvents
	})
	if len(out) > maxFraudDimensionRows {
		out = out[:maxFraudDimensionRows]
	}
	return out
}

func applyDimensionCompareDeltas(current, prior []CustomerFraudByDimensionRowDTO) {
	priorByKey := make(map[string]float64, len(prior))
	for _, row := range prior {
		priorByKey[row.DimensionValue+"|"+row.CampaignID] = float64(row.IVTEvents)
	}
	for i := range current {
		key := current[i].DimensionValue + "|" + current[i].CampaignID
		prev := priorByKey[key]
		delta := 0.0
		if prev > 0 {
			delta = (float64(current[i].IVTEvents) - prev) / prev * 100
		}
		current[i].DeltaLabel = formatDeltaLabel(delta)
		current[i].DeltaTone = deltaTone(delta / 100)
	}
}
