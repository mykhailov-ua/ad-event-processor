package logpipeline

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type RollupInserter interface {
	InsertRollups(ctx context.Context, rows []RollupRow) error
}

type FilterRejectSliceInserter interface {
	InsertFilterRejectSlices(ctx context.Context, rows []FilterRejectSliceRow) error
}

type ClickHouseRollupInserter struct {
	conn driver.Conn
}

func NewClickHouseRollupInserter(conn driver.Conn) *ClickHouseRollupInserter {
	return &ClickHouseRollupInserter{conn: conn}
}

func (inserter *ClickHouseRollupInserter) InsertRollups(ctx context.Context, rows []RollupRow) error {
	if len(rows) == 0 {
		return nil
	}

	batch, err := inserter.conn.PrepareBatch(ctx, `
		INSERT INTO ad_event_processor.audit_log_rollups (
			rollup_hour, campaign_id, event_type,
			event_count, fraud_event_count, billable_event_count,
			sample_click_ids, source_segment, warm_dest_sha256
		)
	`)
	if err != nil {
		return fmt.Errorf("prepare audit_log_rollups batch: %w", err)
	}

	for i := range rows {
		row := &rows[i]
		if err := batch.Append(
			row.RollupHour,
			row.CampaignID,
			row.EventType,
			row.EventCount,
			row.FraudEventCount,
			row.BillableEventCount,
			row.SampleClickIDs,
			row.SourceSegment,
			row.WarmDestSHA256,
		); err != nil {
			return fmt.Errorf("append rollup row: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("send audit_log_rollups batch: %w", err)
	}
	return nil
}

func (inserter *ClickHouseRollupInserter) InsertFilterRejectSlices(ctx context.Context, rows []FilterRejectSliceRow) error {
	if len(rows) == 0 {
		return nil
	}
	batch, err := inserter.conn.PrepareBatch(ctx, `
		INSERT INTO ad_event_processor.filter_reject_slices (
			rollup_hour, reject_kind, placement_id, country, reject_count
		)
	`)
	if err != nil {
		return fmt.Errorf("prepare filter_reject_slices batch: %w", err)
	}
	for i := range rows {
		row := &rows[i]
		if err := batch.Append(row.RollupHour, row.RejectKind, row.PlacementID, row.Country, row.RejectCount); err != nil {
			return fmt.Errorf("append filter reject slice row: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("send filter_reject_slices batch: %w", err)
	}
	return nil
}

type MemoryRollupInserter struct {
	Rows      []RollupRow
	SliceRows []FilterRejectSliceRow
}

func (inserter *MemoryRollupInserter) InsertRollups(_ context.Context, rows []RollupRow) error {
	inserter.Rows = append(inserter.Rows, rows...)
	return nil
}

func (inserter *MemoryRollupInserter) InsertFilterRejectSlices(_ context.Context, rows []FilterRejectSliceRow) error {
	inserter.SliceRows = append(inserter.SliceRows, rows...)
	return nil
}
