package controlplane

import (
	"testing"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/migrationsource"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestImportMigrationCampaigns_invalidPayloadNoCampaignImport_holdout(t *testing.T) {
	t.Parallel()
	svc := &Service{}
	_, err := svc.ImportMigrationCampaigns(t.Context(), campaign.ImportMigrationSpec{
		CustomerID:     uuid.New(),
		IdempotencyKey: "batch-1",
		SourceKind:     migrationsource.SourceKind("unknown-source"),
		Payload:        []byte(`{}`),
	})
	require.Error(t, err)
}

func TestPreviewMigrationPull_invalidPayloadNoPG_holdout(t *testing.T) {
	t.Parallel()
	svc := &Service{}
	_, err := svc.PreviewMigrationPull(t.Context(), campaign.PullMigrationPreviewSpec{
		SourceKind: migrationsource.SourceKind("unknown-source"),
		BaseURL:    "https://example.test",
	})
	require.Error(t, err)
}
