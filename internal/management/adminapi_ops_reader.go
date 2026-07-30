package management

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"espx/internal/adminapi"
	"espx/internal/ingestion"
	db "espx/internal/ingestion/sqlc"
	"espx/internal/management/authz"

	"github.com/google/uuid"
)

type opsReader struct {
	svc *Service
}

func newOpsReader(svc *Service) *opsReader {
	if svc == nil {
		return nil
	}
	return &opsReader{svc: svc}
}

func (r *opsReader) GetIncidentSnapshot(ctx context.Context) (adminapi.IncidentSnapshotDTO, error) {
	report, err := r.svc.GetShardHealth(ctx)
	if err != nil {
		return adminapi.IncidentSnapshotDTO{}, err
	}
	return adminapi.IncidentSnapshotDTO{
		EmergencyBreaker: report.EmergencyBreaker,
		Shards:           mapShardStatuses(report.Shards),
		Outbox: adminapi.OutboxHealthSummary{
			Pending:              report.Outbox.Pending,
			OldestPendingSeconds: report.Outbox.OldestPendingSeconds,
			LastProcessedEventID: report.Outbox.LastProcessedEventID,
		},
		StreamLag:     []adminapi.ShardStreamLag{},
		BreakerStates: map[string]string{},
	}, nil
}

func mapShardStatuses(in []ShardHealthStatus) []adminapi.ShardHealthStatus {
	out := make([]adminapi.ShardHealthStatus, len(in))
	for i, s := range in {
		out[i] = adminapi.ShardHealthStatus{
			ShardID:             s.ShardID,
			PingOK:              s.PingOK,
			PingError:           s.PingError,
			PingLatencyMs:       s.PingLatencyMs,
			ConfigVersion:       s.ConfigVersion,
			ConfigVersionLag:    s.ConfigVersionLag,
			ConfigVersionSynced: s.ConfigVersionSynced,
		}
	}
	return out
}

func (r *opsReader) ListOutboxEvents(ctx context.Context, status, eventType, cursor string, limit int32) (adminapi.OutboxListResult, error) {
	if r.svc.GetPool() == nil {
		return adminapi.OutboxListResult{}, fmt.Errorf("postgres pool not configured")
	}
	if limit <= 0 {
		limit = 50
	}
	var cursorID int64
	if cursor != "" {
		n, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			return adminapi.OutboxListResult{}, errInvalidQuery("invalid cursor")
		}
		cursorID = n
	}
	rows, err := r.svc.GetPool().Query(ctx, `
		SELECT id, event_type, status, created_at
		FROM outbox_events
		WHERE ($1::text = '' OR status = $1)
		  AND ($2::text = '' OR event_type = $2)
		  AND ($3::bigint = 0 OR id < $3)
		ORDER BY id DESC
		LIMIT $4`, status, eventType, cursorID, limit+1)
	if err != nil {
		return adminapi.OutboxListResult{}, err
	}
	defer rows.Close()

	var items []adminapi.OutboxEventDTO
	for rows.Next() {
		var id int64
		var eventTypeVal, statusVal string
		var createdAt time.Time
		if err := rows.Scan(&id, &eventTypeVal, &statusVal, &createdAt); err != nil {
			return adminapi.OutboxListResult{}, err
		}
		items = append(items, adminapi.OutboxEventDTO{
			ID:        id,
			EventType: eventTypeVal,
			Status:    statusVal,
			CreatedAt: createdAt.UTC().Format(time.RFC3339),
		})
	}
	result := adminapi.OutboxListResult{Items: items, Total: int64(len(items))}
	if int32(len(items)) > limit {
		result.Items = items[:limit]
		result.NextCursor = strconv.FormatInt(result.Items[len(result.Items)-1].ID, 10)
	}
	return result, rows.Err()
}

func (r *opsReader) ListDLQEntries(ctx context.Context, cursor string, limit int) (adminapi.FanOutResult[adminapi.DLQEntryDTO], error) {
	_ = ctx
	_ = cursor
	_ = limit
	return adminapi.FanOutResult[adminapi.DLQEntryDTO]{Items: []adminapi.DLQEntryDTO{}}, nil
}

func (r *opsReader) EnqueueDLQRetry(ctx context.Context, payload adminapi.DLQRetryPayload, idempotencyKey string) error {
	_ = ctx
	_ = payload
	_ = idempotencyKey
	return fmt.Errorf("dlq retry not configured")
}

func (r *opsReader) GetShardHealthFanOut(ctx context.Context) (adminapi.ShardHealthAPIResponse, error) {
	report, err := r.svc.GetShardHealth(ctx)
	if err != nil {
		return adminapi.ShardHealthAPIResponse{}, err
	}
	return adminapi.ShardHealthAPIResponse{
		ShardHealthReport: adminapi.ShardHealthReport{
			EmergencyBreaker: report.EmergencyBreaker,
			Outbox: adminapi.OutboxHealthSummary{
				Pending:              report.Outbox.Pending,
				OldestPendingSeconds: report.Outbox.OldestPendingSeconds,
				LastProcessedEventID: report.Outbox.LastProcessedEventID,
			},
			Shards: mapShardStatuses(report.Shards),
		},
	}, nil
}

func (r *opsReader) ExportAuditCSV(ctx context.Context, cursor string, w io.Writer) (adminapi.AuditExportResult, error) {
	var cursorID int64
	if cursor != "" {
		n, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			return adminapi.AuditExportResult{}, errInvalidQuery("invalid cursor")
		}
		cursorID = n
	}
	cw := csv.NewWriter(w)
	if cursorID == 0 {
		_ = cw.Write([]string{"id", "admin_id", "action", "target_type", "target_id", "is_masked", "created_at"})
	}
	rows, err := db.New(r.svc.GetPool()).ListAuditLogsExport(ctx, db.ListAuditLogsExportParams{
		Column1: cursorID,
		Limit:   500,
	})
	if err != nil {
		return adminapi.AuditExportResult{}, err
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
		return adminapi.AuditExportResult{}, err
	}
	byteCount := 0
	if buf, ok := w.(*bytes.Buffer); ok {
		byteCount = buf.Len()
	}
	result := adminapi.AuditExportResult{Bytes: byteCount}
	if len(rows) >= 500 {
		result.Truncated = true
		result.NextCursor = strconv.FormatInt(lastID, 10)
	}
	return result, nil
}

func (r *opsReader) LookupLedgerIDForPaymentIntent(ctx context.Context, intentID string) (string, error) {
	paymentIntentID, err := uuid.Parse(intentID)
	if err != nil {
		return "", err
	}
	row, err := db.New(r.svc.GetPool()).GetLedgerByPaymentIntent(ctx, ingestion.ToUUID(paymentIntentID))
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(row.ID, 10), nil
}

func (r *opsReader) ListReconRuns(ctx context.Context, service string, limit, offset int32) ([]adminapi.ReconRunDTO, int64, error) {
	runs, total, err := r.svc.ListReconRuns(ctx, service, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]adminapi.ReconRunDTO, len(runs))
	for i, run := range runs {
		out[i] = adminapi.ReconRunDTO{
			Service:            run.Service,
			ID:                 run.ID,
			PeriodStart:        run.PeriodStart,
			PeriodEnd:          run.PeriodEnd,
			Status:             run.Status,
			TotalDelta:         run.TotalDelta,
			CampaignsChecked:   run.CampaignsChecked,
			DiscrepanciesFound: run.DiscrepanciesFound,
			FindingsCount:      run.FindingsCount,
			IntentsChecked:     run.IntentsChecked,
			ErrorMessage:       run.ErrorMessage,
			CreatedAt:          run.CreatedAt,
			CompletedAt:        run.CompletedAt,
		}
	}
	return out, total, nil
}

type auditLister struct{ svc *Service }

func (a auditLister) ListAuditLogs(ctx context.Context, limit, offset int32, redactPII bool) (any, int64, error) {
	return a.svc.ListAuditLogsRedacted(ctx, limit, offset, redactPII)
}

type rolesReloader struct{ mw *AuthMiddleware }

func (r rolesReloader) ReloadRoles() error {
	if r.mw == nil || r.mw.policy == nil {
		return fmt.Errorf("policy store not configured")
	}
	return authz.LoadRolesYAML(authz.DefaultRolesPath(), r.mw.policy)
}

func (r rolesReloader) RolesPath() string { return authz.DefaultRolesPath() }
