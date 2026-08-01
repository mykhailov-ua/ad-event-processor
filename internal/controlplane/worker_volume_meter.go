package controlplane

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	billingdb "espx/internal/billing/db"
	"espx/internal/database"
	"espx/internal/licensing"
	"espx/internal/metrics"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	meterBillableEvents = "events"
	meterAcceptedEvents = "accepted_events"
	volumeMeterSourcePG = "pg"
	volumeMeterSourceCH = "ch"
)

type VolumeMeterWorker struct {
	pool     *pgxpool.Pool
	ch       *database.CHQuery
	source   string
	interval time.Duration
	pgGate   *MgmtPgGate
}

func NewVolumeMeterWorker(pool *pgxpool.Pool, ch *database.CHQuery, source string, interval time.Duration, pgGate *MgmtPgGate) *VolumeMeterWorker {
	if interval <= 0 {
		interval = time.Hour
	}
	if source == "" {
		source = volumeMeterSourcePG
	}
	return &VolumeMeterWorker{pool: pool, ch: ch, source: source, interval: interval, pgGate: pgGate}
}

func (w *VolumeMeterWorker) Start(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	if w.source == volumeMeterSourceCH && w.ch == nil {
		slog.Warn("volume meter source=ch but clickhouse query is nil, worker not started")
		return
	}
	slog.Info("volume meter worker starting", "interval", w.interval, "source", w.source)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.RunHour(ctx, time.Now().UTC()); err != nil {
				slog.Error("volume meter rollup failed", "err", err)
			}
		}
	}
}

type rollupRow struct {
	CampaignID uuid.UUID
	EventType  string
	Count      uint64
}

type pgMeterRow struct {
	CustomerID uuid.UUID
	Count      int64
}

func (w *VolumeMeterWorker) RunHour(ctx context.Context, now time.Time) error {
	if w.pgGate != nil {
		if err := w.pgGate.AcquireLow(ctx); err != nil {
			return err
		}
		defer w.pgGate.ReleaseLow()
	}
	return w.runHour(ctx, now)
}

func (w *VolumeMeterWorker) runHour(ctx context.Context, now time.Time) error {
	hourEnd := now.Truncate(time.Hour)
	hourStart := hourEnd.Add(-time.Hour)
	period := time.Date(hourStart.Year(), hourStart.Month(), 1, 0, 0, 0, 0, time.UTC)

	if w.source == volumeMeterSourcePG {
		return w.runPGHour(ctx, hourStart, hourEnd, period)
	}
	return w.runCHHour(ctx, hourStart, hourEnd, period)
}

func (w *VolumeMeterWorker) runPGHour(ctx context.Context, hourStart, hourEnd, period time.Time) error {
	rows, err := w.queryPGRollups(ctx, hourStart, hourEnd)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	q := billingdb.New(w.pool)
	var customers int
	for _, row := range rows {
		if row.Count <= 0 {
			continue
		}
		if _, err := q.IncrementUsageMeter(ctx, billingdb.IncrementUsageMeterParams{
			CustomerID: pgtype.UUID{Bytes: row.CustomerID, Valid: true},
			Meter:      meterAcceptedEvents,
			Period:     pgtype.Date{Time: period, Valid: true},
			Value:      row.Count,
		}); err != nil {
			return fmt.Errorf("increment usage meter customer=%s: %w", row.CustomerID, err)
		}
		customers++
	}
	metrics.VolumeMeterRowsTotal.Add(float64(len(rows)))
	slog.Info("volume meter pg rollup complete",
		"hour", hourStart.Format(time.RFC3339),
		"customers", customers,
		"rows", len(rows),
	)
	return nil
}

func (w *VolumeMeterWorker) queryPGRollups(ctx context.Context, from, to time.Time) ([]pgMeterRow, error) {
	const q = `
		SELECT c.customer_id, COUNT(*)::bigint AS cnt
		FROM events e
		JOIN campaigns c ON c.id = e.campaign_id
		WHERE e.created_at >= $1 AND e.created_at < $2 AND e.status = 'accepted'
		GROUP BY c.customer_id`

	pgRows, err := w.pool.Query(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("pg volume meter query: %w", err)
	}
	defer pgRows.Close()

	var out []pgMeterRow
	for pgRows.Next() {
		var row pgMeterRow
		if err := pgRows.Scan(&row.CustomerID, &row.Count); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, pgRows.Err()
}

func (w *VolumeMeterWorker) runCHHour(ctx context.Context, hourStart, hourEnd, period time.Time) error {
	rows, err := w.queryCHRollups(ctx, hourStart, hourEnd)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	campaignCustomers, err := w.loadCampaignCustomers(ctx)
	if err != nil {
		return err
	}

	customerUnits := ComputeWeightedUnitsFromRows(rows, campaignCustomers)
	q := billingdb.New(w.pool)
	for custID, units := range customerUnits {
		if units <= 0 {
			continue
		}
		if _, err := q.IncrementUsageMeter(ctx, billingdb.IncrementUsageMeterParams{
			CustomerID: pgtype.UUID{Bytes: custID, Valid: true},
			Meter:      meterBillableEvents,
			Period:     pgtype.Date{Time: period, Valid: true},
			Value:      units,
		}); err != nil {
			return fmt.Errorf("increment usage meter customer=%s: %w", custID, err)
		}
	}
	metrics.VolumeMeterRowsTotal.Add(float64(len(customerUnits)))
	slog.Info("volume meter ch rollup complete",
		"hour", hourStart.Format(time.RFC3339),
		"customers", len(customerUnits),
	)
	return nil
}

func (w *VolumeMeterWorker) queryCHRollups(ctx context.Context, from, to time.Time) ([]rollupRow, error) {
	const q = `
		SELECT
			campaign_id,
			event_type,
			sum(event_count) AS cnt
		FROM ad_event_processor.audit_log_rollups
		WHERE rollup_hour >= ? AND rollup_hour < ?
		GROUP BY campaign_id, event_type`

	chRows, err := w.ch.Query(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("clickhouse rollup query: %w", err)
	}
	defer chRows.Close()

	var out []rollupRow
	for chRows.Next() {
		var row rollupRow
		if err := chRows.Scan(&row.CampaignID, &row.EventType, &row.Count); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, chRows.Err()
}

func (w *VolumeMeterWorker) loadCampaignCustomers(ctx context.Context) (map[uuid.UUID]uuid.UUID, error) {
	pgRows, err := w.pool.Query(ctx, `SELECT id, customer_id FROM campaigns`)
	if err != nil {
		return nil, err
	}
	defer pgRows.Close()

	out := make(map[uuid.UUID]uuid.UUID)
	for pgRows.Next() {
		var campID, custID uuid.UUID
		if err := pgRows.Scan(&campID, &custID); err != nil {
			return nil, err
		}
		out[campID] = custID
	}
	return out, pgRows.Err()
}

func ComputeWeightedUnitsFromRows(rows []rollupRow, campaignCustomers map[uuid.UUID]uuid.UUID) map[uuid.UUID]int64 {
	customerUnits := make(map[uuid.UUID]int64)
	for _, row := range rows {
		custID, ok := campaignCustomers[row.CampaignID]
		if !ok {
			continue
		}
		cat := licensing.ClassifyEventType(row.EventType)
		units := int64(row.Count) * licensing.BillableWeightPermille(cat) / 1000
		customerUnits[custID] += units
	}
	return customerUnits
}
