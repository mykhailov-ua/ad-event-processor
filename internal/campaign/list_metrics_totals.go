package campaign

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CampaignListMetricsTotalsResponse struct {
	CampaignCount     int64                     `json:"campaign_count"`
	FlowCount         int64                     `json:"flow_count"`
	MarginBreachCount int64                     `json:"margin_breach_count"`
	Totals            CampaignListMetricsRowDTO `json:"totals"`
	From              string                    `json:"from"`
	To                string                    `json:"to"`
	Stale             bool                      `json:"stale"`
}

func (h *CampaignsHTTPHandlers) listCampaignMetricsTotals(w http.ResponseWriter, r *http.Request) {
	if h.PostgresPool == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "service unavailable")
		return
	}
	filter, err := h.campaignListFilterFromRequest(r)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	from, to, err := parseCampaignListMetricsRange(r)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	resp, err := QueryCampaignListMetricsTotals(
		r.Context(),
		h.PostgresPool,
		h.ClickHouseQuery,
		h.MarginDefaultThresholdBps,
		filter,
		from,
		to,
	)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func QueryCampaignListMetricsTotals(
	ctx context.Context,
	pool *pgxpool.Pool,
	clickhouseQuery *database.ClickHouseQuery,
	defaultThresholdBps int,
	filter ListCampaignsFilter,
	from, to time.Time,
) (CampaignListMetricsTotalsResponse, error) {
	if pool == nil {
		return CampaignListMetricsTotalsResponse{}, ErrServiceUnavailable()
	}
	if !to.After(from) {
		return CampaignListMetricsTotalsResponse{}, fmt.Errorf("%w: to must be after from", ErrInvalidTimeRange)
	}

	q := db.New(pool)
	keyRows, err := q.ListCampaignListKeysForFilter(ctx, CampaignListKeysParamsFromFilter(filter))
	if err != nil {
		return CampaignListMetricsTotalsResponse{}, err
	}
	if err := validateCampaignListExtendedSortKeyCount(len(keyRows)); err != nil {
		return CampaignListMetricsTotalsResponse{}, err
	}

	campaignIDs := make([]uuid.UUID, len(keyRows))
	for i, row := range keyRows {
		campaignIDs[i] = uuid.UUID(row.ID.Bytes)
	}

	flowCount, err := q.CountCampaignFlowsForFilter(ctx, CampaignCountFlowsParamsFromFilter(filter))
	if err != nil {
		return CampaignListMetricsTotalsResponse{}, err
	}

	if len(campaignIDs) == 0 {
		return CampaignListMetricsTotalsResponse{
			CampaignCount:     0,
			FlowCount:         flowCount,
			MarginBreachCount: 0,
			Totals:            CampaignListMetricsRowDTO{},
			From:              from.UTC().Format(time.RFC3339),
			To:                to.UTC().Format(time.RFC3339),
			Stale:             clickhouseQuery != nil,
		}, nil
	}

	var aggregated CampaignListMetricsRowDTO
	var marginBreachCount int64
	stale := false
	for start := 0; start < len(campaignIDs); start += campaignListMetricsMaxIDs {
		end := start + campaignListMetricsMaxIDs
		if end > len(campaignIDs) {
			end = len(campaignIDs)
		}
		chunk := campaignIDs[start:end]
		batch, err := BatchCampaignListMetrics(ctx, pool, clickhouseQuery, defaultThresholdBps, chunk, from, to)
		if err != nil {
			return CampaignListMetricsTotalsResponse{}, err
		}
		if batch.Stale {
			stale = true
		}
		marginBreachCount += countCampaignListMetricsMarginBreaches(batch.Items)
		for _, row := range batch.Items {
			mergeCampaignListMetricsRow(&aggregated, row)
		}
	}
	enrichCampaignListMetricsRowDerived(&aggregated)

	return CampaignListMetricsTotalsResponse{
		CampaignCount:     int64(len(campaignIDs)),
		FlowCount:         flowCount,
		MarginBreachCount: marginBreachCount,
		Totals:            aggregated,
		From:              from.UTC().Format(time.RFC3339),
		To:                to.UTC().Format(time.RFC3339),
		Stale:             stale,
	}, nil
}

func countCampaignListMetricsMarginBreaches(items map[string]CampaignListMetricsRowDTO) int64 {
	var count int64
	for _, row := range items {
		if row.MarginBreach {
			count++
		}
	}
	return count
}

func mergeCampaignListMetricsRow(into *CampaignListMetricsRowDTO, row CampaignListMetricsRowDTO) {
	into.Impressions += row.Impressions
	into.Clicks += row.Clicks
	into.Conversions += row.Conversions
	into.UniqueClicks += row.UniqueClicks
	into.Blocks += row.Blocks
	into.LeadsRaw += row.LeadsRaw
	into.HoldLeads += row.HoldLeads
	into.RejectedLeads += row.RejectedLeads
	into.LPClicks += row.LPClicks
	into.LPViews += row.LPViews
	into.Bots += row.Bots
	into.AdvertiserSpendMicro += row.AdvertiserSpendMicro
	into.RtbCostMicro += row.RtbCostMicro
	into.OperatorMarginMicro += row.OperatorMarginMicro
	into.PublisherPayoutMicro += row.PublisherPayoutMicro
}

func aggregateCampaignListMetricsRows(rows []CampaignListMetricsRowDTO) CampaignListMetricsRowDTO {
	var total CampaignListMetricsRowDTO
	for _, row := range rows {
		mergeCampaignListMetricsRow(&total, row)
	}
	enrichCampaignListMetricsRowDerived(&total)
	return total
}
