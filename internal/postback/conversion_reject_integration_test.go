package postback

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestConversionReject_pendingSkipsPostback_integration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: conversion reject pending postback hold")
	}
	ctx := context.Background()
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'c', 0, 'USD')`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO campaigns (id, name, status, customer_id) VALUES ($1, 'Camp', 'ACTIVE', $2)`, campaignID, customerID)
	require.NoError(t, err)

	q := db.New(pool)
	require.NoError(t, q.UpsertPostbackConfig(ctx, db.UpsertPostbackConfigParams{
		CampaignID:        pgtype.UUID{Bytes: campaignID, Valid: true},
		Provider:          "webhook",
		UrlTemplate:       "https://example.com/pb",
		ApiTokenEncrypted: []byte("x"),
		TargetEvent:       "conversion",
	}))

	cfgReject := config.ConversionReject{
		Enabled:       true,
		RejectNoClick: true,
	}
	applier := NewConversionRejectApplier(cfgReject, nil, nil, nil)
	evt := &domain.Event{
		ClickID:    "clk-pending-1",
		CampaignID: campaignID,
		Type:       "conversion",
		Payload:    []byte(`{"goal_name":"lead","revenue_micro":1000000}`),
	}
	applier.ApplyBatch(ctx, []*domain.Event{evt})
	require.True(t, domain.ConversionValidationPending(evt.Payload))
	require.Empty(t, evt.FraudReason)

	enq := NewConversionPostbackEnqueuer(q)
	enq.OnBatchStored(ctx, []*domain.Event{evt})

	events, err := q.GetPendingPostbackEventsForUpdate(ctx, db.GetPendingPostbackEventsForUpdateParams{
		Limit:   10,
		Column2: 120,
	})
	require.NoError(t, err)
	require.Empty(t, events)
}
