package reportjob

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ExportDeps struct {
	Pool *pgxpool.Pool // non-nil: PG queue + worker; nil keeps in-memory map + per-job goroutine
	// Cold-path writers: reports package queries CH/PG and writes local path (OS boundary).
	WriteReport                   func(ctx context.Context, path string, spec ReportJobSpec) error
	WriteCampaignImportValidation func(ctx context.Context, path string, spec ReportJobSpec) error
}
