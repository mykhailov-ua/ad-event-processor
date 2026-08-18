package ingestion

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func SettingsPGSync(pool *pgxpool.Pool) func(context.Context) (map[string]string, int64, error) {
	if pool == nil {
		return nil
	}
	return func(ctx context.Context) (map[string]string, int64, error) {
		rows, err := pool.Query(ctx, "SELECT key, value FROM system_settings")
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()

		out := make(map[string]string)
		for rows.Next() {
			var key, value string
			if err := rows.Scan(&key, &value); err != nil {
				return nil, 0, err
			}
			out[key] = value
		}
		if err := rows.Err(); err != nil {
			return nil, 0, err
		}

		var version int64
		err = pool.QueryRow(ctx, `
			SELECT COALESCE(MAX(id), 0)
			FROM outbox_events
			WHERE event_type = 'UPDATE_SETTINGS' AND status = 'PROCESSED'
		`).Scan(&version)
		if err != nil {
			return out, 0, nil
		}
		return out, version, nil
	}
}
