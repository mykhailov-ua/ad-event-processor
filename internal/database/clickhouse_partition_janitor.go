package database

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"ad-event-processor/internal/metrics"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type EmergencyDropAlerter func(table, partition string, diskUsedPct float64)

type ClickHouseTableRetention struct {
	Table string
	Days  int
}

type ClickHouseJanitorOptions struct {
	RetentionDays            int
	ExtraTables              []ClickHouseTableRetention
	EmergencyDropPercent     int
	RecompressPartsThreshold int
	OffPeakStartHourUTC      int
	OffPeakEndHourUTC        int
	OnEmergencyDrop          EmergencyDropAlerter
	Now                      func() time.Time
	DiskUsedPercentFn        func(context.Context) (float64, error)
}

type ClickHousePartitionJanitor struct {
	conn                     driver.Conn
	retentionDays            int
	extraTables              []ClickHouseTableRetention
	emergencyDropPercent     int
	recompressPartsThreshold int
	offPeakStartHourUTC      int
	offPeakEndHourUTC        int
	tables                   []string
	onEmergencyDrop          EmergencyDropAlerter
	now                      func() time.Time
	diskUsedPercentFn        func(context.Context) (float64, error)
	wg                       sync.WaitGroup
}

func NewClickHousePartitionJanitor(conn driver.Conn, opts *ClickHouseJanitorOptions) *ClickHousePartitionJanitor {
	retention := opts.RetentionDays
	if retention <= 0 {
		retention = 180
	}
	partsThreshold := opts.RecompressPartsThreshold
	if partsThreshold <= 0 {
		partsThreshold = 8
	}
	startHour := opts.OffPeakStartHourUTC
	if startHour < 0 || startHour > 23 {
		startHour = 2
	}
	endHour := opts.OffPeakEndHourUTC
	if endHour < 0 || endHour > 23 {
		endHour = 6
	}
	nowFn := opts.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	return &ClickHousePartitionJanitor{
		conn:                     conn,
		retentionDays:            retention,
		extraTables:              opts.ExtraTables,
		emergencyDropPercent:     opts.EmergencyDropPercent,
		recompressPartsThreshold: partsThreshold,
		offPeakStartHourUTC:      startHour,
		offPeakEndHourUTC:        endHour,
		tables:                   []string{"impressions", "clicks", "conversions", "fraud_events"},
		onEmergencyDrop:          opts.OnEmergencyDrop,
		now:                      nowFn,
		diskUsedPercentFn:        opts.DiskUsedPercentFn,
	}
}

func (j *ClickHousePartitionJanitor) Run(ctx context.Context) error {
	if j == nil || j.conn == nil {
		return nil
	}

	diskPct, err := j.diskUsedPercent(ctx)
	if err != nil {
		return fmt.Errorf("disk usage: %w", err)
	}
	metrics.ClickHouseDiskUsedPercent.Set(diskPct)
	j.updateActivePartsMax(ctx)

	if j.emergencyDropPercent > 0 && diskPct >= float64(j.emergencyDropPercent) {
		if err := j.runEmergencyDrop(ctx, diskPct); err != nil {
			return err
		}
		return nil
	}

	if err := j.runRetentionDrop(ctx); err != nil {
		return err
	}

	if chOffPeakUTC(j.now(), j.offPeakStartHourUTC, j.offPeakEndHourUTC) {
		if err := j.runRecompress(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (j *ClickHousePartitionJanitor) runRetentionDrop(ctx context.Context) error {
	if err := j.dropTableRetention(ctx, j.tables, j.retentionDays); err != nil {
		return err
	}
	for _, tr := range j.extraTables {
		if tr.Table == "" || tr.Days <= 0 {
			continue
		}
		if err := j.dropTableRetention(ctx, []string{tr.Table}, tr.Days); err != nil {
			return err
		}
	}
	return nil
}

func (j *ClickHousePartitionJanitor) dropTableRetention(ctx context.Context, tables []string, retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	cutoff := j.now().UTC().AddDate(0, 0, -retentionDays)
	cutoffPart, err := strconv.Atoi(cutoff.Format("200601"))
	if err != nil {
		return err
	}

	for _, table := range tables {
		rows, err := j.conn.Query(ctx, `
SELECT partition
FROM system.parts
WHERE active AND database = currentDatabase() AND table = ?
GROUP BY partition`, table)
		if err != nil {
			return fmt.Errorf("list partitions for %s: %w", table, err)
		}
		for rows.Next() {
			var part string
			if err := rows.Scan(&part); err != nil {
				_ = rows.Close()
				return err
			}
			if !partitionOlderThan(part, cutoffPart) {
				continue
			}
			if err := j.dropPartition(ctx, table, part, "retention"); err != nil {
				_ = rows.Close()
				return err
			}
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if err := rows.Err(); err != nil {
			return err
		}
	}
	return nil
}

func (j *ClickHousePartitionJanitor) runEmergencyDrop(ctx context.Context, diskPct float64) error {
	currentPart := j.now().UTC().Format("200601")
	rows, err := j.conn.Query(ctx, `
SELECT table, partition
FROM system.parts
WHERE active
 AND database = currentDatabase()
 AND table IN ('impressions', 'clicks', 'conversions', 'fraud_events')
 AND partition < ?
GROUP BY table, partition
ORDER BY partition ASC
LIMIT 1`, currentPart)
	if err != nil {
		return fmt.Errorf("select emergency drop candidate: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var table, part string
	if !rows.Next() {
		slog.Warn("clickhouse emergency drop: no droppable partition", "disk_used_pct", diskPct)
		return rows.Err()
	}
	if err := rows.Scan(&table, &part); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := j.dropPartition(ctx, table, part, "emergency"); err != nil {
		return err
	}
	metrics.ClickHouseJanitorEmergencyDropTotal.Inc()
	if j.onEmergencyDrop != nil {
		j.onEmergencyDrop(table, part, diskPct)
	}
	slog.Warn("clickhouse emergency partition drop",
		"table", table,
		"partition", part,
		"disk_used_pct", diskPct,
		"threshold_pct", j.emergencyDropPercent,
	)
	return nil
}

func (j *ClickHousePartitionJanitor) runRecompress(ctx context.Context) error {
	rows, err := j.conn.Query(ctx, `
SELECT table, partition, count() AS parts
FROM system.parts
WHERE active
 AND database = currentDatabase()
 AND table IN ('impressions', 'clicks', 'conversions', 'fraud_events')
GROUP BY table, partition
HAVING parts >= ?
ORDER BY parts DESC
LIMIT 1`, j.recompressPartsThreshold)
	if err != nil {
		return fmt.Errorf("select recompress candidate: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var table, part string
	var parts uint64
	if !rows.Next() {
		return rows.Err()
	}
	if err := rows.Scan(&table, &part, &parts); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if !ValidClickHouseIdentifier(table) || !ValidClickHouseIdentifier(part) {
		return fmt.Errorf("invalid clickhouse identifier table=%q partition=%q", table, part)
	}

	stmt := "OPTIMIZE TABLE " + table + " PARTITION '" + part + "' FINAL"
	if err := j.conn.Exec(ctx, stmt); err != nil {
		return fmt.Errorf("recompress %s.%s: %w", table, part, err)
	}
	metrics.ClickHouseJanitorRecompressTotal.Inc()
	slog.Info("clickhouse partition recompressed",
		"table", table,
		"partition", part,
		"parts_before", parts,
	)
	return nil
}

func (j *ClickHousePartitionJanitor) dropPartition(ctx context.Context, table, part, reason string) error {
	if !ValidClickHouseIdentifier(table) || !ValidClickHouseIdentifier(part) {
		return fmt.Errorf("invalid clickhouse identifier table=%q partition=%q", table, part)
	}
	stmt := "ALTER TABLE " + table + " DROP PARTITION '" + part + "'"
	if err := j.conn.Exec(ctx, stmt); err != nil {
		return fmt.Errorf("drop partition %s.%s (%s): %w", table, part, reason, err)
	}
	if reason == "retention" {
		metrics.ClickHouseJanitorRetentionDropTotal.Inc()
	}
	slog.Info("dropped clickhouse partition", "table", table, "partition", part, "reason", reason)
	return nil
}

func (j *ClickHousePartitionJanitor) diskUsedPercent(ctx context.Context) (float64, error) {
	if j.diskUsedPercentFn != nil {
		return j.diskUsedPercentFn(ctx)
	}
	var free, total uint64
	err := j.conn.QueryRow(ctx, `
SELECT sum(free_space), sum(total_space)
FROM system.disks`).Scan(&free, &total)
	if err != nil {
		return 0, err
	}
	if total == 0 {
		return 0, nil
	}
	used := float64(total-free) / float64(total) * 100
	return used, nil
}

func partitionOlderThan(part string, cutoffYYYYMM int) bool {
	partInt, err := strconv.Atoi(part)
	if err != nil {
		return false
	}
	return partInt < cutoffYYYYMM
}

func chOffPeakUTC(now time.Time, startHour, endHour int) bool {
	h := now.UTC().Hour()
	if startHour == endHour {
		return false
	}
	if startHour < endHour {
		return h >= startHour && h < endHour
	}
	return h >= startHour || h < endHour
}

func (j *ClickHousePartitionJanitor) StartBackground(ctx context.Context, interval time.Duration) {
	if j == nil {
		return
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	j.wg.Add(1)
	go func() {
		defer j.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
				if err := j.Run(runCtx); err != nil {
					slog.Error("clickhouse partition janitor failed", "error", err)
				}
				cancel()
			}
		}
	}()
}

func (j *ClickHousePartitionJanitor) Wait() {
	j.wg.Wait()
}

func (j *ClickHousePartitionJanitor) updateActivePartsMax(ctx context.Context) {
	if j == nil || j.conn == nil {
		return
	}
	var maxParts uint64
	err := j.conn.QueryRow(ctx, `
SELECT max(parts) FROM (
 SELECT count() AS parts
 FROM system.parts
 WHERE active AND database = currentDatabase()
 GROUP BY table, partition
)`).Scan(&maxParts)
	if err != nil {
		slog.Warn("clickhouse active parts gauge failed", "error", err)
		return
	}
	metrics.ClickHouseActivePartsMax.Set(float64(maxParts))
}
