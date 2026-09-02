package campaign

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// campaignListMetricsMaxRange caps batch list-metrics windows; matches admin stats range guard.
const campaignListMetricsMaxRange = 90 * 24 * time.Hour

type ListCampaignsFilter struct {
	CustomerID     uuid.UUID
	Status         string
	OwnerUserID    pgtype.UUID
	TargetCountry  string
	BudgetMinMicro pgtype.Int8
	BudgetMaxMicro pgtype.Int8
	Limit          int32
	Offset         int32
}

func ResolveListOwnerUserFilter(ctx context.Context, r *http.Request) pgtype.UUID {
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
