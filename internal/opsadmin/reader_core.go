package opsadmin

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

const (
	defaultOpsMetricScrapeInterval = 15 * time.Second
	defaultOpsMetricRetention      = 24 * time.Hour
	defaultOpsMetricScrapeTimeout  = 5 * time.Second
)

const insertOpsMetricSampleSQL = `INSERT INTO ops.metric_samples (name, labels_hash, ts, value)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (name, labels_hash, ts) DO UPDATE SET value = EXCLUDED.value`

func (r *Reader) GetIncidentSnapshot(ctx context.Context) (IncidentSnapshotDTO, error) {
	report, err := r.deps.GetShardHealth(ctx)
	if err != nil {
		return IncidentSnapshotDTO{}, err
	}
	snap := IncidentSnapshotDTO{
		EmergencyBreaker: report.EmergencyBreaker,
		Shards:           report.Shards,
		Outbox:           report.Outbox,
		StreamLag:        []ShardStreamLag{},
		BreakerStates:    map[string]string{},
	}
	stale, _ := r.incidentDashboardStale(ctx)
	snap.StaleDashboard = stale
	if stale {
		campaigns, err := r.listAffectedCampaigns(ctx, 50)
		if err == nil && len(campaigns) > 0 {
			snap.AffectedCampaigns = campaigns
		}
	}
	return snap, nil
}

func (r *Reader) ListOutboxEvents(ctx context.Context, status, eventType, cursor string, limit int32) (OutboxListResult, error) {
	if r.pool() == nil {
		return OutboxListResult{}, fmt.Errorf("postgres pool not configured")
	}
	if limit <= 0 {
		limit = 50
	}
	var cursorID int64
	if cursor != "" {
		n, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			return OutboxListResult{}, errInvalidQuery("invalid cursor")
		}
		cursorID = n
	}
	rows, err := r.pool().Query(ctx, `
		SELECT id, event_type, status, created_at
		FROM outbox_events
		WHERE ($1::text = '' OR status = $1)
		 AND ($2::text = '' OR event_type = $2)
		 AND ($3::bigint = 0 OR id < $3)
		ORDER BY id DESC
		LIMIT $4`, status, eventType, cursorID, limit+1)
	if err != nil {
		return OutboxListResult{}, err
	}
	defer rows.Close()

	var items []OutboxEventDTO
	for rows.Next() {
		var id int64
		var eventTypeVal, statusVal string
		var createdAt time.Time
		if err := rows.Scan(&id, &eventTypeVal, &statusVal, &createdAt); err != nil {
			return OutboxListResult{}, err
		}
		items = append(items, OutboxEventDTO{
			ID:        id,
			EventType: eventTypeVal,
			Status:    statusVal,
			CreatedAt: createdAt.UTC().Format(time.RFC3339),
		})
	}
	result := OutboxListResult{Items: items, Total: int64(len(items))}
	if int32(len(items)) > limit {
		result.Items = items[:limit]
		result.NextCursor = strconv.FormatInt(result.Items[len(result.Items)-1].ID, 10)
	}
	return result, rows.Err()
}

func (r *Reader) ListDLQEntries(ctx context.Context, cursor string, limit int) (FanOutResult[DLQEntryDTO], error) {
	return r.listDLQEntries(ctx, cursor, limit)
}

func (r *Reader) EnqueueDLQRetry(ctx context.Context, payload DLQRetryPayload, idempotencyKey string) error {
	return r.enqueueDLQRetry(ctx, payload, idempotencyKey)
}

func (r *Reader) GetShardHealthFanOut(ctx context.Context) (ShardHealthAPIResponse, error) {
	report, err := r.deps.GetShardHealth(ctx)
	if err != nil {
		return ShardHealthAPIResponse{}, err
	}
	return ShardHealthAPIResponse{
		ShardHealthReport: report,
	}, nil
}

func (r *Reader) ExportAuditCSV(ctx context.Context, cursor string, redactPII bool, w io.Writer) (AuditExportResult, error) {
	var cursorID int64
	if cursor != "" {
		n, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			return AuditExportResult{}, errInvalidQuery("invalid cursor")
		}
		cursorID = n
	}
	cw := csv.NewWriter(w)
	if cursorID == 0 {
		_ = cw.Write([]string{"id", "admin_id", "action", "target_type", "target_id", "is_masked", "created_at"})
	}
	rows, err := db.New(r.pool()).ListAuditLogsExport(ctx, db.ListAuditLogsExportParams{
		Column1: cursorID,
		Limit:   500,
	})
	if err != nil {
		return AuditExportResult{}, err
	}
	var lastID int64
	for _, row := range rows {
		adminID := ""
		if row.AdminID.Valid {
			adminID = uuid.UUID(row.AdminID.Bytes).String()
		}
		targetID := ""
		if row.TargetID.Valid {
			targetID = uuid.UUID(row.TargetID.Bytes).String()
		}
		createdAt := ""
		if row.CreatedAt.Valid {
			createdAt = row.CreatedAt.Time.UTC().Format(time.RFC3339)
		}
		if redactPII {
			if adminID != "" {
				adminID = "***"
			}
			if targetID != "" {
				targetID = "***"
			}
		}
		_ = cw.Write([]string{
			strconv.FormatInt(row.ID, 10),
			adminID,
			row.Action,
			row.TargetType,
			targetID,
			strconv.FormatBool(row.IsMasked),
			createdAt,
		})
		lastID = row.ID
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return AuditExportResult{}, err
	}
	byteCount := 0
	if buf, ok := w.(*bytes.Buffer); ok {
		byteCount = buf.Len()
	}
	result := AuditExportResult{Bytes: byteCount}
	if len(rows) >= 500 {
		result.Truncated = true
		result.NextCursor = strconv.FormatInt(lastID, 10)
	}
	return result, nil
}

func (r *Reader) LookupLedgerIDForPaymentIntent(ctx context.Context, intentID string) (string, error) {
	paymentIntentID, err := uuid.Parse(intentID)
	if err != nil {
		return "", err
	}
	row, err := db.New(r.pool()).GetLedgerByPaymentIntent(ctx, domain.ToUUID(paymentIntentID))
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(row.ID, 10), nil
}

func (r *Reader) ListReconRuns(ctx context.Context, service string, limit, offset int32) ([]ReconRunDTO, int64, error) {
	return r.deps.ListReconRuns(ctx, service, limit, offset)
}
