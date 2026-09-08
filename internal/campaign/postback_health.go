package campaign

import (
	"context"
	"errors"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const postbackHealthAlertSuccessRate = 95.0

type PostbackHealthRowDTO struct {
	CampaignID      string   `json:"campaign_id"`
	Provider        string   `json:"provider"`
	SuccessRate24h  *float64 `json:"success_rate_24h,omitempty"`
	P95LatencyMs    *int64   `json:"p95_latency_ms,omitempty"`
	LastError       string   `json:"last_error,omitempty"`
	DLQPendingCount int64    `json:"dlq_pending_count"`
	HealthStatus    string   `json:"health_status"`
}

type PostbackHealthResponseDTO struct {
	Rows                      []PostbackHealthRowDTO `json:"rows"`
	AlertThresholdSuccessRate float64                `json:"alert_threshold_success_rate"`
	RunbookPath               string                 `json:"runbook_path,omitempty"`
}

func ListPostbackHealthRows(ctx context.Context, pool *pgxpool.Pool) ([]PostbackHealthRowDTO, error) {
	if pool == nil {
		return nil, errors.New("postgres pool not configured")
	}
	rows, err := pool.Query(ctx, `
SELECT
    c.campaign_id::text,
    c.provider,
    stats.total_24h,
    stats.success_24h,
    stats.p95_latency_ms,
    COALESCE(stats.last_error, ''),
    COALESCE(dlq.dlq_pending_count, 0)::bigint
FROM postback_configs c
LEFT JOIN LATERAL (
    SELECT
        COUNT(*) FILTER (WHERE d.status IN ('SENT', 'FAILED'))::bigint AS total_24h,
        COUNT(*) FILTER (WHERE d.status = 'SENT')::bigint AS success_24h,
        percentile_cont(0.95) WITHIN GROUP (ORDER BY d.latency_ms)
            FILTER (WHERE d.latency_ms IS NOT NULL) AS p95_latency_ms,
        (
            SELECT d2.error_message
            FROM postback_dispatches d2
            WHERE d2.campaign_id = c.campaign_id
              AND d2.status = 'FAILED'
              AND d2.error_message IS NOT NULL
              AND d2.error_message <> ''
            ORDER BY d2.created_at DESC
            LIMIT 1
        ) AS last_error
    FROM postback_dispatches d
    WHERE d.campaign_id = c.campaign_id
      AND d.created_at >= NOW() - INTERVAL '24 hours'
) stats ON true
LEFT JOIN (
    SELECT campaign_id, COUNT(*)::bigint AS dlq_pending_count
    FROM postback_dlq
    WHERE status = 'FAILED'
    GROUP BY campaign_id
) dlq ON dlq.campaign_id = c.campaign_id
ORDER BY c.campaign_id, c.provider`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]PostbackHealthRowDTO, 0, 32)
	for rows.Next() {
		var campaignID, provider, lastError string
		var total24h, success24h, dlqPending int64
		var p95 *float64
		if err := rows.Scan(&campaignID, &provider, &total24h, &success24h, &p95, &lastError, &dlqPending); err != nil {
			return nil, err
		}
		row := PostbackHealthRowDTO{
			CampaignID:      campaignID,
			Provider:        provider,
			LastError:       lastError,
			DLQPendingCount: dlqPending,
		}
		if total24h > 0 {
			rate := math.Round(10000.0*float64(success24h)/float64(total24h)) / 100.0
			row.SuccessRate24h = &rate
		}
		if p95 != nil && !math.IsNaN(*p95) {
			ms := int64(math.Round(*p95))
			if ms >= 0 {
				row.P95LatencyMs = &ms
			}
		}
		row.HealthStatus = postbackHealthStatus(row, total24h)
		out = append(out, row)
	}
	return out, rows.Err()
}

func postbackHealthStatus(row PostbackHealthRowDTO, total24h int64) string {
	if row.DLQPendingCount > 0 || row.LastError != "" {
		return "fail"
	}
	if row.SuccessRate24h != nil && *row.SuccessRate24h < postbackHealthAlertSuccessRate {
		return "fail"
	}
	if total24h == 0 {
		return "warn"
	}
	return "ok"
}

func ListPostbackHealthRowsForCampaigns(ctx context.Context, pool *pgxpool.Pool, campaignIDs []uuid.UUID) ([]PostbackHealthRowDTO, error) {
	all, err := ListPostbackHealthRows(ctx, pool)
	if err != nil {
		return nil, err
	}
	if len(campaignIDs) == 0 {
		return []PostbackHealthRowDTO{}, nil
	}
	allowed := campaignIDSet(campaignIDs)
	out := make([]PostbackHealthRowDTO, 0, len(all))
	for _, row := range all {
		id, err := uuid.Parse(row.CampaignID)
		if err != nil {
			continue
		}
		if _, ok := allowed[id]; ok {
			out = append(out, row)
		}
	}
	return out, nil
}
