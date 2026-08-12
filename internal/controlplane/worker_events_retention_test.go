package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventsRetentionWorker_DeletesOldRows(t *testing.T) {
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	ctx := context.Background()
	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance) VALUES ($1, 'retention', 0)`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, customer_id, budget_limit)
		VALUES ($1, 'retention-camp', 'ACTIVE', $2, 1000000)`, campaignID, customerID)
	require.NoError(t, err)

	oldAt := time.Now().UTC().AddDate(0, 0, -120)
	oldDate := oldAt.Format("2006-01-02")
	_, err = pool.Exec(ctx, `
		INSERT INTO events (click_id, campaign_id, user_id, event_type, created_at, created_date, status)
		VALUES ($1, $2, 'u1', 'click', $3, $4::date, 'accepted')`,
		"retention-old-1", campaignID, oldAt, oldDate)
	require.NoError(t, err)

	recentAt := time.Now().UTC().Add(-time.Hour)
	_, err = pool.Exec(ctx, `
		INSERT INTO events (click_id, campaign_id, user_id, event_type, created_at, created_date, status)
		VALUES ($1, $2, 'u2', 'click', $3, CURRENT_DATE, 'accepted')`,
		"retention-recent-1", campaignID, recentAt)
	require.NoError(t, err)

	worker := NewEventsRetentionWorker(pool, 90)
	deleted := worker.RunOnce(ctx)
	assert.Equal(t, int64(1), deleted)

	var oldCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM events WHERE click_id = $1`, "retention-old-1").Scan(&oldCount)
	require.NoError(t, err)
	assert.Equal(t, 0, oldCount)

	var recentCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM events WHERE click_id = $1`, "retention-recent-1").Scan(&recentCount)
	require.NoError(t, err)
	assert.Equal(t, 1, recentCount)
}
