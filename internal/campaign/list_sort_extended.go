package campaign

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const campaignListSortMetricsChunk = 250

// campaignListExtendedSortMaxKeys caps in-memory extended metric sort (ListCampaignListKeysForFilter).
const campaignListExtendedSortMaxKeys = 5000

func campaignListSortMetricsChunkCount(campaignCount int) int {
	if campaignCount <= 0 {
		return 0
	}
	return (campaignCount + campaignListSortMetricsChunk - 1) / campaignListSortMetricsChunk
}

// campaignListSortMetricsPGRoundTrips counts PG queries inside loadCampaignListSortMetrics.
func campaignListSortMetricsPGRoundTrips(campaignCount int, customerScoped bool) int {
	if campaignCount <= 0 {
		return 0
	}
	if customerScoped {
		return 2
	}
	return campaignListSortMetricsChunkCount(campaignCount) * 2
}

// campaignListSortMetricsCHRoundTrips counts ClickHouse queries inside loadCampaignListSortMetrics.
func campaignListSortMetricsCHRoundTrips(campaignCount int) int {
	if campaignCount <= 0 {
		return 0
	}
	return campaignListSortMetricsChunkCount(campaignCount) * 2
}

// campaignListExtendedSortPGRoundTrips includes ListCampaignListKeysForFilter plus metrics loads.
func campaignListExtendedSortPGRoundTrips(campaignCount int, customerScoped bool) int {
	return 1 + campaignListSortMetricsPGRoundTrips(campaignCount, customerScoped)
}

func validateCampaignListExtendedSortKeyCount(keyCount int) error {
	if keyCount > campaignListExtendedSortMaxKeys {
		return invalidQueryError(fmt.Sprintf(
			"too many campaigns for metric sort (max %d); narrow filters",
			campaignListExtendedSortMaxKeys,
		))
	}
	return nil
}

type campaignListSortKey struct {
	id   uuid.UUID
	name string
}

type campaignListSortMetrics struct {
	impressions   int64
	clicks        int64
	conversions   int64
	uniqueClicks  int64
	blocks        int64
	rtbCostMicro  int64
	revenueMicro  int64
	profitMicro   int64
	leadsRaw      int64
	holdLeads     int64
	rejectedLeads int64
	lpClicks      int64
	lpViews       int64
	bots          int64
}

func (m *campaignListSortMetrics) syncLeadsRaw() {
	if m.leadsRaw > 0 {
		return
	}
	raw := m.conversions + m.holdLeads + m.rejectedLeads
	if raw > 0 {
		m.leadsRaw = raw
		return
	}
	m.leadsRaw = m.conversions
}

func IsCampaignListExtendedMetricSortField(field string) bool {
	switch strings.TrimSpace(field) {
	case "unique_clicks", "blocks", "cost", "revenue", "profit", "roi",
		"ctr", "cr", "cpc", "cpa", "ecpa", "epc", "cpm", "block_pct",
		"leads", "approved", "hold_leads", "rejected_leads", "approve_rate",
		"lp_clicks", "lp_views", "lp_ctr", "bots", "bot_pct":
		return true
	default:
		return false
	}
}

func IsCampaignListMetricWindowSortField(field string) bool {
	return IsCampaignListStatsSortField(field) || IsCampaignListExtendedMetricSortField(field)
}

func (m campaignListSortMetrics) compareValue(field string) float64 {
	switch strings.TrimSpace(field) {
	case "unique_clicks":
		return float64(m.uniqueClicks)
	case "blocks":
		return float64(m.blocks)
	case "cost":
		return float64(m.rtbCostMicro)
	case "revenue":
		return float64(m.revenueMicro)
	case "profit":
		return float64(m.profitMicro)
	case "ctr":
		if m.impressions <= 0 {
			return 0
		}
		return float64(m.clicks) / float64(m.impressions)
	case "cr":
		if m.clicks <= 0 {
			return 0
		}
		return float64(m.conversions) / float64(m.clicks)
	case "block_pct":
		if m.clicks <= 0 {
			return 0
		}
		return float64(m.blocks) / float64(m.clicks)
	case "cpc":
		if m.clicks <= 0 {
			return 0
		}
		return float64(m.rtbCostMicro) / float64(m.clicks)
	case "cpa", "ecpa":
		if m.conversions <= 0 {
			return 0
		}
		return float64(m.rtbCostMicro) / float64(m.conversions)
	case "epc":
		if m.clicks <= 0 {
			return 0
		}
		return float64(m.revenueMicro) / float64(m.clicks)
	case "cpm":
		if m.impressions <= 0 || m.rtbCostMicro <= 0 {
			return 0
		}
		return float64(m.rtbCostMicro) * 1000 / float64(m.impressions)
	case "roi":
		if m.rtbCostMicro <= 0 {
			return 0
		}
		return float64(m.profitMicro) / float64(m.rtbCostMicro)
	case "leads":
		m.syncLeadsRaw()
		return float64(m.leadsRaw)
	case "approved":
		return float64(m.conversions)
	case "hold_leads":
		return float64(m.holdLeads)
	case "rejected_leads":
		return float64(m.rejectedLeads)
	case "approve_rate":
		m.syncLeadsRaw()
		if m.leadsRaw <= 0 {
			return 0
		}
		return float64(m.conversions) / float64(m.leadsRaw)
	case "lp_clicks":
		return float64(m.lpClicks)
	case "lp_views":
		if m.lpViews > 0 {
			return float64(m.lpViews)
		}
		return float64(m.impressions)
	case "lp_ctr":
		if m.clicks <= 0 {
			return 0
		}
		return float64(m.lpClicks) / float64(m.clicks)
	case "bots":
		return float64(m.bots)
	case "bot_pct":
		if m.clicks <= 0 {
			return 0
		}
		return float64(m.bots) / float64(m.clicks)
	default:
		return 0
	}
}

func ListCampaignPageByExtendedMetricSort(
	ctx context.Context,
	pool *pgxpool.Pool,
	clickhouseQuery *database.ClickHouseQuery,
	filter ListCampaignsFilter,
) ([]uuid.UUID, int64, error) {
	if pool == nil {
		return nil, 0, ErrServiceUnavailable()
	}
	if !filter.StatsRangeSet {
		return nil, 0, invalidQueryError("from and to required for metric sort")
	}
	from := filter.StatsRangeFrom.UTC()
	to := filter.StatsRangeTo.UTC()
	if !to.After(from) {
		return nil, 0, fmt.Errorf("%w: to must be after from", ErrInvalidTimeRange)
	}

	q := db.New(pool)
	keyRows, err := q.ListCampaignListKeysForFilter(ctx, CampaignListKeysParamsFromFilter(filter))
	if err != nil {
		return nil, 0, err
	}
	if len(keyRows) == 0 {
		return nil, 0, nil
	}
	if err := validateCampaignListExtendedSortKeyCount(len(keyRows)); err != nil {
		return nil, 0, err
	}

	keys := make([]campaignListSortKey, len(keyRows))
	ids := make([]uuid.UUID, len(keyRows))
	for i, row := range keyRows {
		id := uuid.UUID(row.ID.Bytes)
		keys[i] = campaignListSortKey{id: id, name: row.Name}
		ids[i] = id
	}

	sortField := strings.TrimSpace(filter.SortField)

	metricsByID, err := loadCampaignListSortMetrics(
		ctx,
		pool,
		clickhouseQuery,
		filter.CustomerID,
		ids,
		from,
		to,
		sortField,
	)
	if err != nil {
		return nil, 0, err
	}

	desc := strings.EqualFold(strings.TrimSpace(filter.SortOrder), "desc")
	sort.SliceStable(keys, func(i, j int) bool {
		left := metricsByID[keys[i].id].compareValue(sortField)
		right := metricsByID[keys[j].id].compareValue(sortField)
		if math.Abs(left-right) < 1e-12 {
			if desc {
				return strings.ToLower(keys[i].name) > strings.ToLower(keys[j].name)
			}
			return strings.ToLower(keys[i].name) < strings.ToLower(keys[j].name)
		}
		if desc {
			return left > right
		}
		return left < right
	})

	total := int64(len(keys))
	offset := int64(filter.Offset)
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		return nil, total, nil
	}
	limit := int64(filter.Limit)
	if limit <= 0 {
		limit = 50
	}
	end := offset + limit
	if end > total {
		end = total
	}
	page := keys[offset:end]
	pageIDs := make([]uuid.UUID, len(page))
	for i, key := range page {
		pageIDs[i] = key.id
	}
	return pageIDs, total, nil
}

func loadCampaignListSortMetrics(
	ctx context.Context,
	pool *pgxpool.Pool,
	clickhouseQuery *database.ClickHouseQuery,
	customerID uuid.UUID,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	sortField string,
) (map[uuid.UUID]campaignListSortMetrics, error) {
	out := make(map[uuid.UUID]campaignListSortMetrics, len(campaignIDs))
	idSet := make(map[uuid.UUID]struct{}, len(campaignIDs))
	for _, id := range campaignIDs {
		out[id] = campaignListSortMetrics{}
		idSet[id] = struct{}{}
	}
	if len(campaignIDs) == 0 {
		return out, nil
	}

	statsFrom, statsTo := CampaignListPGStatsDates(from, to)
	marginStart, marginEnd := CampaignListMarginWindow(from, to)

	q := db.New(pool)
	if customerID != uuid.Nil {
		statsRows, err := q.SumCustomerCampaignStatsInRange(ctx, db.SumCustomerCampaignStatsInRangeParams{
			CustomerID: domain.ToUUID(customerID),
			FromDate:   statsFrom,
			ToDate:     statsTo,
		})
		if err != nil {
			return nil, err
		}
		for _, row := range statsRows {
			id := uuid.UUID(row.CampaignID.Bytes)
			if _, ok := idSet[id]; !ok {
				continue
			}
			entry := out[id]
			entry.impressions = row.Impressions
			entry.clicks = row.Clicks
			entry.conversions = row.Conversions
			out[id] = entry
		}
		marginRows, err := q.SumCustomerMarginWindowInRange(ctx, db.SumCustomerMarginWindowInRangeParams{
			CustomerID:  domain.ToUUID(customerID),
			WindowStart: pgtype.Timestamp{Time: marginStart, Valid: true},
			WindowEnd:   pgtype.Timestamp{Time: marginEnd, Valid: true},
		})
		if err != nil {
			return nil, err
		}
		for _, row := range marginRows {
			id := uuid.UUID(row.CampaignID.Bytes)
			if _, ok := idSet[id]; !ok {
				continue
			}
			entry := out[id]
			entry.rtbCostMicro = row.RtbCostMicro
			entry.profitMicro = row.OperatorMarginMicro
			entry.revenueMicro = row.AdvertiserSpendMicro + row.OperatorMarginMicro
			out[id] = entry
		}
	} else {
		for start := 0; start < len(campaignIDs); start += campaignListSortMetricsChunk {
			end := start + campaignListSortMetricsChunk
			if end > len(campaignIDs) {
				end = len(campaignIDs)
			}
			chunk := campaignIDs[start:end]
			pgIDs := make([]pgtype.UUID, len(chunk))
			for i, id := range chunk {
				pgIDs[i] = domain.ToUUID(id)
			}
			statsRows, err := q.SumCampaignStatsByCampaignIDsInRange(ctx, db.SumCampaignStatsByCampaignIDsInRangeParams{
				CampaignIds: pgIDs,
				FromDate:    statsFrom,
				ToDate:      statsTo,
			})
			if err != nil {
				return nil, err
			}
			for _, row := range statsRows {
				id := uuid.UUID(row.CampaignID.Bytes)
				entry := out[id]
				entry.impressions = row.Impressions
				entry.clicks = row.Clicks
				entry.conversions = row.Conversions
				out[id] = entry
			}
			marginRows, err := q.SumCampaignMarginWindowByCampaignIDsInRange(ctx, db.SumCampaignMarginWindowByCampaignIDsInRangeParams{
				CampaignIds: pgIDs,
				WindowStart: pgtype.Timestamp{Time: marginStart, Valid: true},
				WindowEnd:   pgtype.Timestamp{Time: marginEnd, Valid: true},
			})
			if err != nil {
				return nil, err
			}
			for _, row := range marginRows {
				id := uuid.UUID(row.CampaignID.Bytes)
				entry := out[id]
				entry.rtbCostMicro = row.RtbCostMicro
				entry.profitMicro = row.OperatorMarginMicro
				entry.revenueMicro = row.AdvertiserSpendMicro + row.OperatorMarginMicro
				out[id] = entry
			}
		}
	}

	if clickhouseQuery != nil {
		chCtx, cancel := context.WithTimeout(ctx, campaignListMetricsCHTimeout)
		defer cancel()
		var chErrs []error
		for start := 0; start < len(campaignIDs); start += campaignListSortMetricsChunk {
			end := start + campaignListSortMetricsChunk
			if end > len(campaignIDs) {
				end = len(campaignIDs)
			}
			chunk := campaignIDs[start:end]
			chMetrics, err := reports.QueryCampaignListCHSortMetricsCH(chCtx, clickhouseQuery, chunk, from, to)
			if err != nil {
				chErrs = append(chErrs, err)
				continue
			}
			for campaignID, metrics := range chMetrics {
				id, parseErr := uuid.Parse(campaignID)
				if parseErr != nil {
					continue
				}
				entry := out[id]
				entry.uniqueClicks = metrics.UniqueClicks
				entry.blocks = metrics.Blocks
				entry.lpClicks = metrics.LPClicks
				entry.bots = metrics.Bots
				out[id] = entry
			}
			funnelMetrics, err := reports.QueryCampaignListFunnelMetricsCH(chCtx, clickhouseQuery, chunk, from, to)
			if err != nil {
				chErrs = append(chErrs, err)
				continue
			}
			for campaignID, funnel := range funnelMetrics {
				id, parseErr := uuid.Parse(campaignID)
				if parseErr != nil {
					continue
				}
				entry := out[id]
				entry.holdLeads = funnel.HoldLeads
				entry.rejectedLeads = funnel.RejectedLeads
				if funnel.LeadsRaw() > 0 {
					entry.leadsRaw = funnel.LeadsRaw()
				}
				out[id] = entry
			}
		}
		if len(chErrs) > 0 && CampaignListSortNeedsCHMetrics(sortField) {
			return nil, fmt.Errorf("clickhouse sort metrics: %w", errors.Join(chErrs...))
		}
	}
	for id, entry := range out {
		entry.lpViews = entry.impressions
		entry.syncLeadsRaw()
		out[id] = entry
	}
	return out, nil
}

func OrderCampaignDTOsByIDs(items []CampaignDTO, order []uuid.UUID) []CampaignDTO {
	if len(order) == 0 || len(items) == 0 {
		return items
	}
	byID := make(map[string]CampaignDTO, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	out := make([]CampaignDTO, 0, len(order))
	for _, id := range order {
		if item, ok := byID[id.String()]; ok {
			out = append(out, item)
		}
	}
	return out
}
