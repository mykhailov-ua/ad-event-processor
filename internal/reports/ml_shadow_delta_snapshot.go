package reports

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	mlShadowDeltaSnapshotKey        = "current"
	mlShadowDeltaSnapshotStaleAfter = 24 * time.Hour
)

type MLShadowDeltaSnapshot struct {
	RangeFrom   time.Time
	RangeTo     time.Time
	GeneratedAt time.Time
	Rows        []map[string]any
}

func UpsertMLShadowDeltaSnapshot(ctx context.Context, pool *pgxpool.Pool, snap MLShadowDeltaSnapshot) error {
	if pool == nil {
		return errors.New("postgres unavailable")
	}
	rowsJSON, err := json.Marshal(snap.Rows)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
INSERT INTO ml_shadow_delta_snapshots (snapshot_key, range_from, range_to, rows, generated_at)
VALUES ($1, $2, $3, $4::jsonb, $5)
ON CONFLICT (snapshot_key) DO UPDATE SET
	range_from = EXCLUDED.range_from,
	range_to = EXCLUDED.range_to,
	rows = EXCLUDED.rows,
	generated_at = EXCLUDED.generated_at`,
		mlShadowDeltaSnapshotKey,
		snap.RangeFrom.UTC(),
		snap.RangeTo.UTC(),
		rowsJSON,
		snap.GeneratedAt.UTC(),
	)
	return err
}

func LoadMLShadowDeltaSnapshot(ctx context.Context, pool *pgxpool.Pool) (MLShadowDeltaSnapshot, bool, error) {
	if pool == nil {
		return MLShadowDeltaSnapshot{}, false, nil
	}
	var snap MLShadowDeltaSnapshot
	var rowsJSON []byte
	err := pool.QueryRow(ctx, `
SELECT range_from, range_to, rows, generated_at
FROM ml_shadow_delta_snapshots
WHERE snapshot_key = $1`, mlShadowDeltaSnapshotKey).
		Scan(&snap.RangeFrom, &snap.RangeTo, &rowsJSON, &snap.GeneratedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return MLShadowDeltaSnapshot{}, false, nil
	}
	if err != nil {
		return MLShadowDeltaSnapshot{}, false, err
	}
	if len(rowsJSON) > 0 {
		if err := json.Unmarshal(rowsJSON, &snap.Rows); err != nil {
			return MLShadowDeltaSnapshot{}, false, err
		}
	}
	return snap, true, nil
}

func MLShadowDeltaSnapshotFreshness(snap MLShadowDeltaSnapshot, now time.Time) DataFreshnessDTO {
	dto := DataFreshnessDTO{
		AsOf:        snap.GeneratedAt.UTC().Format(time.RFC3339),
		Consistency: "snapshot",
	}
	if snap.GeneratedAt.IsZero() || now.Sub(snap.GeneratedAt) > mlShadowDeltaSnapshotStaleAfter {
		dto.Stale = true
	}
	return dto
}

func PaginateMLShadowDeltaSnapshotRows(rows []map[string]any, limit, offset int) ([]map[string]any, int64) {
	total := int64(len(rows))
	if offset >= len(rows) {
		return []map[string]any{}, total
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	out := make([]map[string]any, end-offset)
	copy(out, rows[offset:end])
	return out, total
}
