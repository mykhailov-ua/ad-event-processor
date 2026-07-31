package controlplane

import (
	"context"
	"testing"
	"time"

	"espx/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVolumeMeterWorker_PGRunHour(t *testing.T) {
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance) VALUES ($1, 'meter', 0)`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, customer_id, budget_limit)
		VALUES ($1, 'meter-camp', 'ACTIVE', $2, 1000000)`, campaignID, customerID)
	require.NoError(t, err)

	hourStart := time.Now().UTC().Truncate(time.Hour).Add(-time.Hour)
	_, err = pool.Exec(ctx, `
		INSERT INTO events (click_id, campaign_id, user_id, event_type, created_at, created_date, status)
		VALUES ($1, $2, 'u1', 'click', $3, $4::date, 'accepted')`,
		"meter-click-1", campaignID, hourStart.Add(10*time.Minute), hourStart.Format("2006-01-02"))
	require.NoError(t, err)

	w := NewVolumeMeterWorker(pool, nil, volumeMeterSourcePG, time.Hour, nil)
	require.NoError(t, w.RunHour(ctx, hourStart.Add(time.Hour)))

	var value int64
	err = pool.QueryRow(ctx, `
		SELECT value FROM billing.usage_meters
		WHERE customer_id = $1 AND meter = $2 AND period = date_trunc('month', $3::timestamptz)::date`,
		customerID, meterAcceptedEvents, hourStart).Scan(&value)
	require.NoError(t, err)
	assert.Equal(t, int64(1), value)
}
