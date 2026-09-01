package fraudadmin

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func normalizeFraudLabelLimit(limit int) int {
	if limit <= 0 {
		return ManualLabelsDefaultLimit
	}
	if limit > ManualLabelsMaxLimit {
		return ManualLabelsMaxLimit
	}
	return limit
}

func normalizeFraudLabelOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func (l *Labels) ListMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]MLManualLabelDTO, int64, error) {
	if customerID == uuid.Nil {
		return nil, 0, ValidationError("customer_id is required")
	}
	if l == nil || l.host == nil || l.host.LabelsPool() == nil {
		return nil, 0, fmt.Errorf("postgres pool not configured")
	}
	limit = normalizeFraudLabelLimit(limit)
	offset = normalizeFraudLabelOffset(offset)

	var total int64
	if err := l.host.LabelsPool().QueryRow(ctx, `
		SELECT count(*)
		FROM ml_manual_labels
		WHERE customer_id = $1`, domain.ToUUID(customerID)).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count ml_manual_labels: %w", err)
	}

	rows, err := l.host.LabelsPool().Query(ctx, `
		SELECT ip_hash, label, reason, source, created_at
		FROM ml_manual_labels
		WHERE customer_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, domain.ToUUID(customerID), limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query ml_manual_labels: %w", err)
	}
	defer rows.Close()

	out := make([]MLManualLabelDTO, 0, limit)
	for rows.Next() {
		var row MLManualLabelDTO
		var createdAt time.Time
		if err := rows.Scan(&row.IPHash, &row.Label, &row.Reason, &row.Source, &createdAt); err != nil {
			return nil, 0, err
		}
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		row.CreatedAtDisplay = coldpath.RFC3339Display(row.CreatedAt)
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (l *Labels) UpsertMLManualLabelForCustomer(ctx context.Context, customerID uuid.UUID, ipHash string, label int, reason string) error {
	if customerID == uuid.Nil {
		return ValidationError("customer_id is required")
	}
	if err := ValidateMLIPHash(ipHash); err != nil {
		return err
	}
	if label != 0 && label != 1 {
		return ValidationError("label must be 0 or 1")
	}
	if l == nil || l.host == nil || l.host.LabelsPool() == nil {
		return fmt.Errorf("postgres pool not configured")
	}
	_, err := l.host.LabelsPool().Exec(ctx, `
		INSERT INTO ml_manual_labels (ip_hash, label, reason, source, customer_id, created_at)
		VALUES ($1, $2, $3, 'admin_ui', $4, NOW())
		ON CONFLICT (ip_hash) DO UPDATE SET
			label = EXCLUDED.label,
			reason = EXCLUDED.reason,
			source = EXCLUDED.source,
			customer_id = EXCLUDED.customer_id,
			created_at = NOW()`,
		ipHash, label, reason, domain.ToUUID(customerID))
	return err
}

func (l *Labels) BulkUpsertMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, rows []FraudManualLabelRow) (int, error) {
	if customerID == uuid.Nil {
		return 0, ValidationError("customer_id is required")
	}
	if len(rows) == 0 {
		return 0, ValidationError("rows required")
	}
	if len(rows) > ManualLabelsBulkMax {
		return 0, ValidationError(fmt.Sprintf("max %d rows per bulk request", ManualLabelsBulkMax))
	}
	if l == nil || l.host == nil || l.host.LabelsPool() == nil {
		return 0, fmt.Errorf("postgres pool not configured")
	}

	var inserted int
	err := pgx.BeginFunc(ctx, l.host.LabelsPool(), func(tx pgx.Tx) error {
		batch := &pgx.Batch{}
		for i, row := range rows {
			if err := ValidateMLIPHash(row.IPHash); err != nil {
				return fmt.Errorf("row %d: %w", i+1, err)
			}
			if row.Label != 0 && row.Label != 1 {
				return fmt.Errorf("row %d: label must be 0 or 1", i+1)
			}
			batch.Queue(`
				INSERT INTO ml_manual_labels (ip_hash, label, reason, source, customer_id, created_at)
				VALUES ($1, $2, $3, 'admin_ui', $4, NOW())
				ON CONFLICT (ip_hash) DO UPDATE SET
					label = EXCLUDED.label,
					reason = EXCLUDED.reason,
					source = EXCLUDED.source,
					customer_id = EXCLUDED.customer_id,
					created_at = NOW()`,
				row.IPHash, row.Label, row.Reason, domain.ToUUID(customerID))
		}
		br := tx.SendBatch(ctx, batch)
		for range rows {
			if _, err := br.Exec(); err != nil {
				_ = br.Close()
				return err
			}
			inserted++
		}
		return br.Close()
	})
	if err != nil {
		return 0, err
	}
	return inserted, nil
}
