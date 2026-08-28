package campaign

import (
	"encoding/json"

	"ad-event-processor/internal/migrationsource"

	"github.com/google/uuid"
)

type ImportMigrationSpec struct {
	CustomerID       uuid.UUID
	IdempotencyKey   string
	SourceKind       migrationsource.SourceKind
	Payload          []byte
	NamePrefix       string
	BudgetLimitMicro *int64
}

type ImportMigrationFailure struct {
	Ref     string `json:"ref"`
	Name    string `json:"name,omitempty"`
	Message string `json:"message"`
}

type ImportMigrationResult struct {
	ImportBatchID string                    `json:"import_batch_id"`
	Imported      []ImportCampaignResult    `json:"imported"`
	Warnings      []migrationsource.Warning `json:"warnings,omitempty"`
	Failed        []ImportMigrationFailure  `json:"failed,omitempty"`
}

type PullMigrationPreviewSpec struct {
	SourceKind migrationsource.SourceKind
	BaseURL    string
	APIToken   string
	PullPath   string
}

type PullMigrationImportSpec struct {
	PullMigrationPreviewSpec
	CustomerID       uuid.UUID
	IdempotencyKey   string
	NamePrefix       string
	BudgetLimitMicro *int64
}

type MigratePreviewRequest struct {
	SourceKind string          `json:"source_kind"`
	Payload    json.RawMessage `json:"payload"`
}

type MigrateImportRequest struct {
	CustomerID       string          `json:"customer_id"`
	SourceKind       string          `json:"source_kind"`
	Payload          json.RawMessage `json:"payload"`
	NamePrefix       string          `json:"name_prefix,omitempty"`
	BudgetLimitMicro *int64          `json:"budget_limit_micro,omitempty"`
}

type MigratePullRequest struct {
	SourceKind       string `json:"source_kind"`
	BaseURL          string `json:"base_url"`
	APIToken         string `json:"api_token"`
	PullPath         string `json:"pull_path,omitempty"`
	CustomerID       string `json:"customer_id,omitempty"`
	NamePrefix       string `json:"name_prefix,omitempty"`
	BudgetLimitMicro *int64 `json:"budget_limit_micro,omitempty"`
}

type ImportValidateJobRequest struct {
	CustomerID string          `json:"customer_id"`
	SourceKind string          `json:"source_kind"`
	Payload    json.RawMessage `json:"payload"`
}
