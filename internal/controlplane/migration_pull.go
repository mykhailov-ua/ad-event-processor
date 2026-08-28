package controlplane

import (
	"context"

	"ad-event-processor/internal/migrationsource"
)

func (s *Service) PreviewMigrationPull(ctx context.Context, spec PullMigrationPreviewSpec) (migrationsource.PreviewResult, error) {
	payload, err := migrationsource.FetchRemotePayload(ctx, migrationsource.PullSpec{
		SourceKind: spec.SourceKind,
		BaseURL:    spec.BaseURL,
		APIToken:   spec.APIToken,
		PullPath:   spec.PullPath,
	})
	if err != nil {
		return migrationsource.PreviewResult{}, errValidation(err.Error())
	}
	return migrationsource.Preview(spec.SourceKind, payload, nil)
}

func (s *Service) ImportMigrationPull(ctx context.Context, spec PullMigrationImportSpec) (ImportMigrationResult, error) {
	payload, err := migrationsource.FetchRemotePayload(ctx, migrationsource.PullSpec{
		SourceKind: spec.SourceKind,
		BaseURL:    spec.BaseURL,
		APIToken:   spec.APIToken,
		PullPath:   spec.PullPath,
	})
	if err != nil {
		return ImportMigrationResult{}, errValidation(err.Error())
	}
	return s.ImportMigrationCampaigns(ctx, ImportMigrationSpec{
		CustomerID:       spec.CustomerID,
		IdempotencyKey:   spec.IdempotencyKey,
		SourceKind:       spec.SourceKind,
		Payload:          payload,
		NamePrefix:       spec.NamePrefix,
		BudgetLimitMicro: spec.BudgetLimitMicro,
	})
}
