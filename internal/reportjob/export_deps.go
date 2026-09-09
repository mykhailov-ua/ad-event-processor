package reportjob

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ExportDeps struct {
	Pool *pgxpool.Pool // non-nil: PG queue + worker; nil keeps in-memory map + per-job goroutine
	// Deployment export chunk cap (bytes); used to clamp row_limit server-side.
	ExportChunkMaxBytes func(ctx context.Context) int
	// Optional catalog hook; when nil, only deployment chunk bytes cap row_limit.
	ReportLicenseGated func(reportKey string) bool
	// Cold-path writers: reports package queries ClickHouse/Postgres and writes local path (OS boundary).
	WriteReport                   func(ctx context.Context, path string, spec ReportJobSpec) error
	WriteCampaignImportValidation func(ctx context.Context, path string, spec ReportJobSpec) error
}
