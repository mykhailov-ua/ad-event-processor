package reportjob

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ExportDeps struct {
	Pool                          *pgxpool.Pool
	WriteReport                   func(ctx context.Context, path string, spec ReportJobSpec) error
	WriteCampaignImportValidation func(ctx context.Context, path string, spec ReportJobSpec) error
}
