package campaign

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ad-event-processor/internal/domain"

	"ad-event-processor/internal/teamscope"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// campaignListMetricsMaxRange caps batch list-metrics windows; matches admin stats range guard.
const campaignListMetricsMaxRange = 90 * 24 * time.Hour

type ListCampaignsFilter struct {
	CustomerID     uuid.UUID
	Status         string
	WarningsOnly   bool
	OwnerUserID    pgtype.UUID
	OwnerUserIDs   []uuid.UUID
	TargetCountry  string
	BudgetMinMicro pgtype.Int8
	BudgetMaxMicro pgtype.Int8
	SearchQuery    string
	PacingMode     string
	SortField      string
	SortOrder      string
	StatsFrom      pgtype.Date
	StatsTo        pgtype.Date
	StatsRangeFrom time.Time
	StatsRangeTo   time.Time
	StatsRangeSet  bool
	Limit          int32
	Offset         int32
}

func IsCampaignListStatsSortField(field string) bool {
	switch strings.TrimSpace(field) {
	case "clicks", "impressions", "conversions":
		return true
	default:
		return false
	}
}

// ApplyListScopeFilter merges team scope with optional client owner_user_id override.
func ApplyListScopeFilter(ctx context.Context, pool *pgxpool.Pool, filter *ListCampaignsFilter) error {
	if filter == nil {
		return nil
	}
	queryOwner := filter.OwnerUserID
	scope, err := teamscope.ResolveListScope(ctx, pool, filter.CustomerID)
	if err != nil {
		return err
	}
	filter.OwnerUserID = scope.OwnerUserID
	filter.OwnerUserIDs = scope.OwnerUserIDs
	if queryOwner.Valid && teamscope.AllowOwnerQueryOverride(ctx) {
		filter.OwnerUserID = queryOwner
		filter.OwnerUserIDs = nil
	}
	return nil
}

func ResolveListOwnerUserFilter(ctx context.Context, r *http.Request) pgtype.UUID {
	if !allowOwnerQueryOverride(ctx) {
		return pgtype.UUID{}
	}
	scoped := CampaignOwnerUserFilter(ctx)
	if scoped.Valid {
		return scoped
	}
	raw := strings.TrimSpace(r.URL.Query().Get("owner_user_id"))
	if raw == "" {
		return pgtype.UUID{}
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return pgtype.UUID{}
	}
	return domain.ToUUID(id)
}

func parseOptionalBudgetMicroQuery(r *http.Request, key string) pgtype.Int8 {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return pgtype.Int8{}
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: value, Valid: true}
}

func parseTargetCountryQuery(r *http.Request) string {
	return strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))
}

// parseCampaignListMetricsRange reads from/to (RFC3339). Defaults: last 7d, to truncated to UTC hour.
func parseCampaignListMetricsRange(r *http.Request) (from, to time.Time, err error) {
	now := time.Now().UTC().Truncate(time.Hour)
	to = now
	from = now.Add(-7 * 24 * time.Hour)

	if toStr := strings.TrimSpace(r.URL.Query().Get("to")); toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			return time.Time{}, time.Time{}, invalidQueryError("invalid to timestamp")
		}
		to = to.UTC()
	}
	if fromStr := strings.TrimSpace(r.URL.Query().Get("from")); fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, invalidQueryError("invalid from timestamp")
		}
		from = from.UTC()
	}
	if !to.After(from) {
		return time.Time{}, time.Time{}, invalidQueryError("to must be after from")
	}
	if to.Sub(from) > campaignListMetricsMaxRange {
		return time.Time{}, time.Time{}, invalidQueryError(fmt.Sprintf("time range exceeds %d days", int(campaignListMetricsMaxRange/(24*time.Hour))))
	}
	return from, to, nil
}
