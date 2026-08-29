package costsync

import (
	"context"
	"errors"
	"time"

	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (w *Worker) reconcileCampaigns(ctx context.Context, lines []CostLine, date time.Time) error {
	seen := make(map[uuid.UUID]struct{})
	for _, line := range lines {
		if line.LineType != LineTypeSpend {
			continue
		}
		seen[line.CampaignID] = struct{}{}
	}
	if len(seen) == 0 {
		return nil
	}

	campaignIDs := make([]uuid.UUID, 0, len(seen))
	for campID := range seen {
		campaignIDs = append(campaignIDs, campID)
	}

	rows, err := w.pool.Query(ctx, `
		SELECT
			c.id,
			c.customer_id,
			COALESCE(cc.api_spend, 0) AS api_spend,
			COALESCE(tr.tracker_spend, 0) AS tracker_spend
		FROM campaigns c
		LEFT JOIN (
			SELECT campaign_id, SUM(amount_usd_micro)::bigint AS api_spend
			FROM campaign_costs
			WHERE cost_date = $1 AND line_type = 'spend'
			GROUP BY campaign_id
		) cc ON cc.campaign_id = c.id
		LEFT JOIN (
			SELECT campaign_id,
				COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0)::bigint AS tracker_spend
			FROM balance_ledger
			WHERE created_at >= $1::date
			 AND created_at < ($1::date + INTERVAL '1 day')
			 AND type IN ('FEE', 'RECONCILIATION_ADJUST', 'REFUND')
			GROUP BY campaign_id
		) tr ON tr.campaign_id = c.id
		WHERE c.id = ANY($2)
		 AND COALESCE(cc.api_spend, 0) != COALESCE(tr.tracker_spend, 0)`, date, campaignIDs)
	if err != nil {
		return err
	}
	defer rows.Close()

	batch := &pgx.Batch{}
	var adjustments int
	for rows.Next() {
		var campID, customerID uuid.UUID
		var apiSpend, trackerSpend int64
		if err := rows.Scan(&campID, &customerID, &apiSpend, &trackerSpend); err != nil {
			return err
		}
		delta := apiSpend - trackerSpend
		hash := reconciliationHash(customerID, campID, date)
		batch.Queue(`
			INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, idempotency_hash)
			VALUES ($1, $2, $3, 'RECONCILIATION_ADJUST', $4)
			ON CONFLICT (idempotency_hash) DO NOTHING`,
			pgtype.UUID{Bytes: customerID, Valid: true},
			pgtype.UUID{Bytes: campID, Valid: true},
			-delta,
			hash,
		)
		adjustments++
		metrics.CostSyncReconciliationDelta.Add(float64(abs64(delta)))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if batch.Len() == 0 {
		return nil
	}
	br := w.pool.SendBatch(ctx, batch)
	for i := range adjustments {
		if _, err := br.Exec(); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			_ = br.Close()
			return err
		}
	}
	return br.Close()
}
