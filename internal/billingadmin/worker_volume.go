package billingadmin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const incrementUsageMeterSQL = `
INSERT INTO billing.usage_meters (customer_id, meter, period, value)
VALUES ($1, $2, $3, $4)
ON CONFLICT (customer_id, meter, period) DO UPDATE
SET value = billing.usage_meters.value + EXCLUDED.value`

type PostgresLowGate interface {
	AcquireLow(ctx context.Context) error
	ReleaseLow()
}

const (
	VolumeMeterSourcePG = "pg"
	VolumeMeterSourceCH = "clickhouseQuery"
	meterBillableEvents = "events"
	meterAcceptedEvents = "accepted_events"
)

const MeterAcceptedEvents = meterAcceptedEvents

type VolumeMeterWorker struct {
	pool            *pgxpool.Pool
	clickhouseQuery *database.ClickHouseQuery
	source          string
	interval        time.Duration
	postgresGate    PostgresLowGate
}

func NewVolumeMeterWorker(pool *pgxpool.Pool, clickhouseQuery *database.ClickHouseQuery, source string, interval time.Duration, postgresGate PostgresLowGate) *VolumeMeterWorker {
	if interval <= 0 {
		interval = time.Hour
	}
	if source == "" {
		source = VolumeMeterSourcePG
	}
	return &VolumeMeterWorker{pool: pool, clickhouseQuery: clickhouseQuery, source: source, interval: interval, postgresGate: postgresGate}
}

func (w *VolumeMeterWorker) Start(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	if w.source == VolumeMeterSourceCH && w.clickhouseQuery == nil {
		slog.Warn("volume meter source=clickhouseQuery but clickhouse query is nil, worker not started")
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

type RollupRow struct {
	CampaignID uuid.UUID
	EventType  string
	Count      uint64
}

type postgresMeterRow struct {
	CustomerID uuid.UUID
	Count      int64
}

func (w *VolumeMeterWorker) RunHour(ctx context.Context, now time.Time) error {
	if w.postgresGate != nil {
		if err := w.postgresGate.AcquireLow(ctx); err != nil {
			return err
		}
		defer w.postgresGate.ReleaseLow()
	}
	return w.runHour(ctx, now)
}

func (w *VolumeMeterWorker) runHour(ctx context.Context, now time.Time) error {
	hourEnd := now.Truncate(time.Hour)
	hourStart := hourEnd.Add(-time.Hour)
	period := time.Date(hourStart.Year(), hourStart.Month(), 1, 0, 0, 0, 0, time.UTC)

	if w.source == VolumeMeterSourcePG {
		return w.runPGHour(ctx, hourStart, hourEnd, period)
	}
	return w.runClickHouseHour(ctx, hourStart, hourEnd, period)
}

func (w *VolumeMeterWorker) runPGHour(ctx context.Context, hourStart, hourEnd, period time.Time) error {
	rows, err := w.queryPostgresRollups(ctx, hourStart, hourEnd)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	units := mapFromPGMeterRows(rows)
	if err := batchIncrementUsageMeters(ctx, w.pool, meterAcceptedEvents, period, units); err != nil {
		return err
	}
	metrics.VolumeMeterRowsTotal.Add(float64(len(rows)))
	slog.Info("volume meter pg rollup complete",
		"hour", hourStart.Format(time.RFC3339),
		"customers", len(units),
		"rows", len(rows),
	)
	return nil
}

func mapFromPGMeterRows(rows []postgresMeterRow) map[uuid.UUID]int64 {
	out := make(map[uuid.UUID]int64, len(rows))
	for _, row := range rows {
		if row.Count > 0 {
			out[row.CustomerID] = row.Count
		}
	}
	return out
}

func batchIncrementUsageMeters(ctx context.Context, pool *pgxpool.Pool, meter string, period time.Time, units map[uuid.UUID]int64) error {
	if len(units) == 0 {
		return nil
	}
	postgresPeriod := pgtype.Date{Time: period, Valid: true}
	batch := &pgx.Batch{}
	for custID, value := range units {
		if value <= 0 {
			continue
		}
		batch.Queue(incrementUsageMeterSQL, pgtype.UUID{Bytes: custID, Valid: true}, meter, postgresPeriod, value)
	}
	if batch.Len() == 0 {
		return nil
	}
	br := pool.SendBatch(ctx, batch)
	defer func() { _ = br.Close() }()
	for i := range batch.Len() {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("increment usage meter batch item %d: %w", i, err)
		}
	}
	return nil
}

func (w *VolumeMeterWorker) queryPostgresRollups(ctx context.Context, from, to time.Time) ([]postgresMeterRow, error) {
	const q = `
		SELECT c.customer_id, COUNT(*)::bigint AS cnt
		FROM events e
		JOIN campaigns c ON c.id = e.campaign_id
		WHERE e.created_at >= $1 AND e.created_at < $2 AND e.status = 'accepted'
		GROUP BY c.customer_id`

	postgresRows, err := w.pool.Query(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("pg volume meter query: %w", err)
	}
	defer postgresRows.Close()

	var out []postgresMeterRow
	for postgresRows.Next() {
		var row postgresMeterRow
		if err := postgresRows.Scan(&row.CustomerID, &row.Count); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, postgresRows.Err()
}

func (w *VolumeMeterWorker) runClickHouseHour(ctx context.Context, hourStart, hourEnd, period time.Time) error {
	rows, err := w.queryClickHouseRollups(ctx, hourStart, hourEnd)
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
	if err := batchIncrementUsageMeters(ctx, w.pool, meterBillableEvents, period, customerUnits); err != nil {
		return err
	}
	metrics.VolumeMeterRowsTotal.Add(float64(len(customerUnits)))
	slog.Info("volume meter clickhouseQuery rollup complete",
		"hour", hourStart.Format(time.RFC3339),
		"customers", len(customerUnits),
	)
	return nil
}

func (w *VolumeMeterWorker) queryClickHouseRollups(ctx context.Context, from, to time.Time) ([]RollupRow, error) {
	const q = `
		SELECT
			campaign_id,
			event_type,
			sum(event_count) AS cnt
		FROM ad_event_processor.audit_log_rollups
		WHERE rollup_hour >= ? AND rollup_hour < ?
		GROUP BY campaign_id, event_type`

	clickhouseRows, err := w.clickhouseQuery.Query(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("clickhouse rollup query: %w", err)
	}
	defer func() { _ = clickhouseRows.Close() }()

	var out []RollupRow
	for clickhouseRows.Next() {
		var row RollupRow
		if err := clickhouseRows.Scan(&row.CampaignID, &row.EventType, &row.Count); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, clickhouseRows.Err()
}

func (w *VolumeMeterWorker) loadCampaignCustomers(ctx context.Context) (map[uuid.UUID]uuid.UUID, error) {
	postgresRows, err := w.pool.Query(ctx, `SELECT id, customer_id FROM campaigns`)
	if err != nil {
		return nil, err
	}
	defer postgresRows.Close()

	out := make(map[uuid.UUID]uuid.UUID)
	for postgresRows.Next() {
		var campID, custID uuid.UUID
		if err := postgresRows.Scan(&campID, &custID); err != nil {
			return nil, err
		}
		out[campID] = custID
	}
	return out, postgresRows.Err()
}

func ComputeWeightedUnitsFromRows(rows []RollupRow, campaignCustomers map[uuid.UUID]uuid.UUID) map[uuid.UUID]int64 {
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
