package coldpath

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const espxMigrationsTable = "public.ad_event_processor_migrations"

func ApplyTrackedSchemaMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if pool == nil {
		return fmt.Errorf("coldpath: nil pool")
	}
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS public.ad_event_processor_migrations (
			filename   TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`); err != nil {
		return fmt.Errorf("ensure %s: %w", espxMigrationsTable, err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s: %w", dir, err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		var applied bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM public.ad_event_processor_migrations WHERE filename = $1)
		`, entry.Name()).Scan(&applied)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if applied {
			continue
		}

		sqlBytes, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		upSQL := gooseUpSQL(string(sqlBytes))
		if strings.TrimSpace(upSQL) == "" {
			continue
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, upSQL); err != nil {
			_ = tx.Rollback(ctx)
			if migrationAlreadyApplied(err) {
				if _, recErr := pool.Exec(ctx, `
					INSERT INTO public.ad_event_processor_migrations (filename) VALUES ($1)
					ON CONFLICT (filename) DO NOTHING
				`, entry.Name()); recErr != nil {
					return fmt.Errorf("record skipped migration %s: %w", entry.Name(), recErr)
				}
				continue
			}
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO public.ad_event_processor_migrations (filename) VALUES ($1)
		`, entry.Name()); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func migrationAlreadyApplied(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	switch pgErr.Code {
	case "42P06", "42P07", "42701", "42710", "42723":
		return true
	default:
		return false
	}
}

func gooseUpSQL(sql string) string {
	parts := strings.Split(sql, "-- +goose Down")
	upPart := parts[0]
	upPart = strings.ReplaceAll(upPart, "-- +goose Up", "")
	upPart = strings.ReplaceAll(upPart, "-- +goose StatementBegin", "")
	upPart = strings.ReplaceAll(upPart, "-- +goose StatementEnd", "")
	return upPart
}
