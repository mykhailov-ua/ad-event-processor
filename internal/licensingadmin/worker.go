package licensingadmin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	revokeQueueBatchSize    = 32
	defaultRevokePoll       = 30 * time.Second
	defaultWorkerBatchLimit = 2 * time.Minute
)

type RevokeQueueWorker struct {
	pool       *pgxpool.Pool
	interval   time.Duration
	reload     func(context.Context) error
	currentKey func() string
}

func NewRevokeQueueWorker(pool *pgxpool.Pool, interval time.Duration, reload func(context.Context) error, currentKey func() string) *RevokeQueueWorker {
	if interval <= 0 {
		interval = defaultRevokePoll
	}
	return &RevokeQueueWorker{
		pool:       pool,
		interval:   interval,
		reload:     reload,
		currentKey: currentKey,
	}
}

func (w *RevokeQueueWorker) Start(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	slog.Info("license revoke queue worker starting", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := w.ProcessOnce(ctx)
			if err != nil && ctx.Err() == nil {
				slog.Error("license revoke queue worker failed", "err", err)
				continue
			}
			if n > 0 {
				slog.Info("license revoke queue processed", "count", n)
			}
		}
	}
}

func (w *RevokeQueueWorker) ProcessOnce(ctx context.Context) (int, error) {
	if w == nil || w.pool == nil {
		return 0, nil
	}

	opCtx, cancel := context.WithTimeout(ctx, defaultWorkerBatchLimit)
	defer cancel()

	var keys []string
	var processed int
	err := pgx.BeginFunc(opCtx, w.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(opCtx, `
			SELECT id, license_key
			FROM vendor.license_revoke_queue
			WHERE processed_at IS NULL
			ORDER BY id
			LIMIT $1
			FOR UPDATE SKIP LOCKED`, revokeQueueBatchSize)
		if err != nil {
			return fmt.Errorf("list license revoke queue: %w", err)
		}

		type pendingRow struct {
			id         int64
			licenseKey string
		}
		var pending []pendingRow
		for rows.Next() {
			var row pendingRow
			if err := rows.Scan(&row.id, &row.licenseKey); err != nil {
				rows.Close()
				return fmt.Errorf("scan license revoke queue: %w", err)
			}
			pending = append(pending, row)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}

		for _, row := range pending {
			if _, err := tx.Exec(opCtx, `
				UPDATE vendor.licenses
				SET revoked = TRUE, updated_at = NOW()
				WHERE license_key = $1`, row.licenseKey); err != nil {
				return fmt.Errorf("revoke vendor license %q: %w", row.licenseKey, err)
			}
			tag, err := tx.Exec(opCtx, `
				UPDATE vendor.license_revoke_queue
				SET processed_at = NOW()
				WHERE id = $1 AND processed_at IS NULL`, row.id)
			if err != nil {
				return fmt.Errorf("mark license revoke queue id=%d: %w", row.id, err)
			}
			if tag.RowsAffected() == 0 {
				continue
			}
			keys = append(keys, row.licenseKey)
			processed++
		}
		return nil
	})
	if err != nil {
		return processed, err
	}

	if processed == 0 || w.reload == nil {
		return processed, nil
	}

	current := ""
	if w.currentKey != nil {
		current = w.currentKey()
	}
	for _, key := range keys {
		if current != "" && key != current {
			continue
		}
		if err := w.reload(opCtx); err != nil {
			return processed, fmt.Errorf("reload license after revoke %q: %w", key, err)
		}
		break
	}
	return processed, nil
}
