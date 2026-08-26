package controlplane

import (
	"context"
	"fmt"
	"strings"

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

func (s *Service) ImportMigrationCampaigns(ctx context.Context, spec ImportMigrationSpec) (ImportMigrationResult, error) {
	if s == nil || s.pool == nil {
		return ImportMigrationResult{}, fmt.Errorf("service unavailable")
	}
	if spec.CustomerID == uuid.Nil {
		return ImportMigrationResult{}, errValidation("customer_id is required")
	}
	batchKey := strings.TrimSpace(spec.IdempotencyKey)
	if batchKey == "" {
		return ImportMigrationResult{}, errValidation("idempotency key is required")
	}
	preview, err := migrationsource.Preview(spec.SourceKind, spec.Payload, nil)
	if err != nil {
		return ImportMigrationResult{}, errValidation(err.Error())
	}
	out := ImportMigrationResult{
		ImportBatchID: batchKey,
		Warnings:      preview.Warnings,
	}
	defaultBudget := migrationsource.DefaultMigrateBudgetMicro()
	if spec.BudgetLimitMicro != nil && *spec.BudgetLimitMicro > 0 {
		defaultBudget = *spec.BudgetLimitMicro
	}
	for i, mapped := range preview.MappedCampaigns {
		shape := migrationsource.MappedToExportShape(mapped, spec.NamePrefix, defaultBudget)
		bundle := exportBundleFromMigrationShape(shape)
		itemKey := migrationsource.ImportIdempotencyKey(batchKey, i)
		result, err := s.ImportCampaign(ctx, ImportCampaignSpec{
			CustomerID:     spec.CustomerID,
			IdempotencyKey: itemKey,
			Bundle:         bundle,
		})
		if err != nil {
			out.Failed = append(out.Failed, ImportMigrationFailure{
				Ref:     mapped.Ref,
				Name:    mapped.Name,
				Message: err.Error(),
			})
			continue
		}
		out.Imported = append(out.Imported, result)
	}
	if len(out.Imported) == 0 && len(out.Failed) > 0 {
		return out, errValidation("no campaigns imported")
	}
	if len(out.Imported) > 0 {
		s.AuditLog(ctx, nil, uuid.Nil, "MIGRATE_IMPORT", "migration_batch", nil, auditMigrateImportChange{
			SourceKind: string(spec.SourceKind),
			CustomerID: spec.CustomerID.String(),
			Imported:   len(out.Imported),
			Failed:     len(out.Failed),
		}, auditIdempotencyMeta{IdempotencyKey: batchKey})
	}
	return out, nil
}

func exportBundleFromMigrationShape(shape migrationsource.ExportCampaignShape) CampaignExportBundle {
	camp := CampaignExportCampaign{
		Name:              shape.Name,
		BudgetLimitMicro:  shape.BudgetLimitMicro,
		TargetURL:         shape.TargetURL,
		TrafficTemplateID: shape.TrafficTemplateID,
		ClickQueryParams:  shape.ClickQueryParams,
	}
	bundle := CampaignExportBundle{
		ExportVersion: campaignExportVersion,
		Campaign:      camp,
	}
	if shape.IntegrationSchema != "" {
		bundle.IntegrationSchemaName = shape.IntegrationSchema
	}
	if shape.IngressCostParam != "" {
		camp.IngressCostConfig = &IngressCostConfigDTO{
			Param:  shape.IngressCostParam,
			Scale:  "decimal",
			Policy: "ignore",
		}
		bundle.Campaign = camp
	}
	if shape.PostbackURLTemplate != "" {
		bundle.PostbackConfig = &CampaignExportPostback{
			Provider:    "custom",
			URLTemplate: shape.PostbackURLTemplate,
			TargetEvent: "conversion",
		}
	}
	return bundle
}

type auditMigrateImportChange struct {
	SourceKind string `json:"source_kind"`
	CustomerID string `json:"customer_id"`
	Imported   int    `json:"imported_count"`
	Failed     int    `json:"failed_count"`
}
