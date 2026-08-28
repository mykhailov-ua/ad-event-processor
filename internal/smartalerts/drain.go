package smartalerts

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func CheckStuckDrainJobs(ctx context.Context, host Host) {
	if host == nil || host.Pool() == nil {
		return
	}
	thresholdSec := host.DrainStuckThresholdSec()
	if thresholdSec <= 0 {
		thresholdSec = 900
	}
	jobs, err := listStuckDrainJobs(ctx, host, time.Duration(thresholdSec)*time.Second)
	if err != nil {
		slog.Error("failed to list stuck drain jobs", "error", err)
		return
	}
	for _, job := range jobs {
		host.AlertDrainStuck(ctx, job.version, job.slot, job.state, job.lastError, job.updatedAt)
	}
}

func listStuckDrainJobs(ctx context.Context, host Host, olderThan time.Duration) ([]stuckDrainJob, error) {
	rows, err := host.Pool().Query(ctx, `
		SELECT version, slot, state::text, COALESCE(last_error, ''), updated_at
		FROM redis_slot_migration
		WHERE state IN ('draining', 'failed')
		 AND updated_at < NOW() - $1::interval
		ORDER BY updated_at ASC
	`, fmt.Sprintf("%d seconds", int(olderThan.Seconds())))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []stuckDrainJob
	for rows.Next() {
		var job stuckDrainJob
		if err := rows.Scan(&job.version, &job.slot, &job.state, &job.lastError, &job.updatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

type stuckDrainJob struct {
	version   int32
	slot      int16
	state     string
	lastError string
	updatedAt time.Time
}

func formatOptionalUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}
