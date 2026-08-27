package notify

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/notify/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetentionJanitor_DeletesOldRows(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	svc := newTestService(pool)
	_, err := sendTestNotification(ctx, svc, NotificationInput{
		Provider:  string(db.NotifierProviderTELEGRAM),
		Recipient: "123",
		Title:     "old",
		Body:      "retention test",
	})
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		UPDATE notify.notifications
		SET status = 'SENT', created_at = NOW() - interval '40 days'
		WHERE status = 'PENDING'`)
	require.NoError(t, err)

	janitor := NewRetentionJanitor(pool, time.Hour, 30, 90)
	janitor.runOnce(ctx)

	var remaining int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM notify.notifications`).Scan(&remaining)
	require.NoError(t, err)
	assert.Equal(t, 0, remaining)
}
