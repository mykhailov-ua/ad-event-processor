package stream

import (
	"context"
	"testing"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/budget"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/integrationschema"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestConversionPayoutApplier_integration_processorBatch_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: conversion payout applier batch with postgres mappings")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	t.Cleanup(cleanupDB)
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	t.Cleanup(cleanupRedis)

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'cust', 0, 'USD')`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, status, customer_id, pacing_mode, daily_budget, timezone, current_spend)
		VALUES ($1, 'conv-map', 1000000, 'ACTIVE', $2, 'ASAP', 0, 'UTC', 0)`, campaignID, customerID)
	require.NoError(t, err)

	_, parsed, err := integrationschema.ParseDocument([]byte(`{
		"version": 1,
		"status_map": {"sale": "approved"}
	}`))
	require.NoError(t, err)
	mappings, err := campaign.ConversionMappingsFromStatusSchema(parsed.(*integrationschema.StatusMappingSchema))
	require.NoError(t, err)
	_, err = campaign.ReplaceCampaignConversionMappings(ctx, pool, campaignID, mappings)
	require.NoError(t, err)

	applier := NewConversionPayoutApplier(db.New(pool))
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campaignID,
		Payload:    []byte(`{"status":"sale","revenue_micro":0}`),
	}
	applier.ApplyBatch(ctx, []*domain.Event{evt})
	require.Contains(t, string(evt.Payload), `"goal_name":"approved"`)

	budget.AssertBudgetInvariant(t, ctx, pool, redisClient, campaignID)
}
