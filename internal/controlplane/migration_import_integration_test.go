package controlplane

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/migrationsource"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportMigrationCampaigns_keitaro_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: keitaro migration import creates campaign with click preset")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, nil)
	defer svc.Close()

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Migrate Customer", 500_000_000, "USD"))

	schemaID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO integration_schemas (id, name, version, kind, body)
		VALUES ($1, 'traffic_facebook', 1, 'inbound_tokens', '{}')`, schemaID)
	require.NoError(t, err)

	raw, err := os.ReadFile(filepath.Join("..", "migrationsource", "testdata", "keitaro_facebook_campaign.json"))
	require.NoError(t, err)

	result, err := svc.ImportMigrationCampaigns(ctx, campaign.ImportMigrationSpec{
		CustomerID:     custID,
		IdempotencyKey: "migrate-import-idem-1",
		SourceKind:     migrationsource.SourceKindKeitaroJSON,
		Payload:        raw,
	})
	if err != nil {
		for _, fail := range result.Failed {
			t.Logf("import failed: ref=%s name=%s msg=%s", fail.Ref, fail.Name, fail.Message)
		}
	}
	require.NoError(t, err)
	require.Len(t, result.Imported, 1)

	importedID, err := uuid.Parse(result.Imported[0].ID)
	require.NoError(t, err)

	got, err := svc.GetCampaign(ctx, importedID)
	require.NoError(t, err)
	assert.Equal(t, "meta-facebook", got.TrafficTemplateID)
	assert.Equal(t, "{{campaign.id}}", got.ClickQueryParams["sub2"])

	dup, err := svc.ImportMigrationCampaigns(ctx, campaign.ImportMigrationSpec{
		CustomerID:     custID,
		IdempotencyKey: "migrate-import-idem-1",
		SourceKind:     migrationsource.SourceKindKeitaroJSON,
		Payload:        raw,
	})
	require.NoError(t, err)
	assert.Len(t, dup.Imported, 1)
	assert.Equal(t, result.Imported[0].ID, dup.Imported[0].ID)

	var auditCount int
	err = pool.QueryRow(ctx, `
		SELECT count(*) FROM admin_audit_log
		WHERE action = 'MIGRATE_IMPORT' AND metadata::text LIKE '%migrate-import-idem-1%'`).Scan(&auditCount)
	require.NoError(t, err)
	assert.Equal(t, 1, auditCount)

	var tokenLen int
	err = pool.QueryRow(ctx, `
		SELECT COALESCE(length(api_token_encrypted), 0)
		FROM postback_configs
		WHERE campaign_id = $1`, domain.ToUUID(importedID)).Scan(&tokenLen)
	require.NoError(t, err)
	assert.Equal(t, 0, tokenLen, "migrate import must not copy foreign postback secrets")
}

func TestImportMigrationCampaigns_keitaroStreams_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: keitaro streams migration import round-trip")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, nil)
	defer svc.Close()

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Streams Customer", 500_000_000, "USD"))

	schemaID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO integration_schemas (id, name, version, kind, body)
		VALUES ($1, 'traffic_facebook', 1, 'inbound_tokens', '{}')`, schemaID)
	require.NoError(t, err)

	raw, err := os.ReadFile(filepath.Join("..", "migrationsource", "testdata", "keitaro_facebook_streams.json"))
	require.NoError(t, err)

	result, err := svc.ImportMigrationCampaigns(ctx, campaign.ImportMigrationSpec{
		CustomerID:     custID,
		IdempotencyKey: "migrate-streams-idem-1",
		SourceKind:     migrationsource.SourceKindKeitaroJSON,
		Payload:        raw,
	})
	require.NoError(t, err)
	require.Len(t, result.Imported, 1)

	var warnFound bool
	for _, w := range result.Warnings {
		if w.Slug == "stream_node_unmapped" {
			warnFound = true
		}
	}
	assert.True(t, warnFound)

	importedID, err := uuid.Parse(result.Imported[0].ID)
	require.NoError(t, err)

	got, err := svc.GetCampaign(ctx, importedID)
	require.NoError(t, err)
	assert.Equal(t, "meta-facebook", got.TrafficTemplateID)

	importRow, err := svc.GetCampaignRow(ctx, importedID)
	require.NoError(t, err)
	require.True(t, importRow.FlowID.Valid)

	var importPaths json.RawMessage
	err = pool.QueryRow(ctx, `SELECT paths FROM flows WHERE id = $1`, uuid.UUID(importRow.FlowID.Bytes)).Scan(&importPaths)
	require.NoError(t, err)
	var parsedPaths []campaign.FlowPathDTO
	require.NoError(t, json.Unmarshal(importPaths, &parsedPaths))
	require.Len(t, parsedPaths, 2)
	assert.Equal(t, int32(60), parsedPaths[0].Weight)
	assert.Equal(t, int32(40), parsedPaths[1].Weight)
}
