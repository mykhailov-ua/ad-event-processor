package notify

import (
	"espx/pkg/faultproof"

	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"espx/internal/notify/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t testing.TB) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("notifier_test_db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("secure_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(20*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %s", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %s", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to connect to db: %s", err)
	}

	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Join(filepath.Dir(filename), "..", "..")
	notifierMigrationsDir := filepath.Join(baseDir, "internal/notify/migrations")
	applyMigrations(t, pool, notifierMigrationsDir)

	return pool, func() {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
	}
}

func applyMigrations(t testing.TB, pool *pgxpool.Pool, dir string) {
	t.Helper()
	ctx := context.Background()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read migrations dir %s: %s", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		sqlBytes, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("failed to read migration %s: %s", entry.Name(), err)
		}

		sql := string(sqlBytes)
		parts := strings.Split(sql, "-- +goose Down")
		upPart := parts[0]
		upPart = strings.ReplaceAll(upPart, "-- +goose Up", "")
		upPart = strings.ReplaceAll(upPart, "-- +goose StatementBegin", "")
		upPart = strings.ReplaceAll(upPart, "-- +goose StatementEnd", "")

		if _, err := pool.Exec(ctx, upPart); err != nil {
			t.Fatalf("failed to apply migration %s: %s", entry.Name(), err)
		}
	}
}

func TestService_enqueueAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	svc := newTestService(pool)
	ctx := context.Background()

	req := NotificationInput{
		Provider:  string(db.NotifierProviderTELEGRAM),
		Recipient: "12345678",
		Title:     "Test Alert",
		Body:      "This is a test notification",
	}

	result, err := sendTestNotification(ctx, svc, req)
	require.NoError(t, err)
	assert.NotEmpty(t, result.NotificationID)
	assert.Equal(t, db.NotifierNotificationStatusPENDING, result.Status)

	notification, err := getTestNotification(ctx, svc, result.NotificationID)
	require.NoError(t, err)
	assert.Equal(t, result.NotificationID, notification.ID)
	assert.Equal(t, string(db.NotifierProviderTELEGRAM), notification.Provider)
	assert.Equal(t, "12345678", notification.Recipient)
	assert.Equal(t, "Test Alert", notification.Title)
	assert.Equal(t, "This is a test notification", notification.Body)
	assert.Equal(t, db.NotifierNotificationStatusPENDING, notification.Status)
	assert.NotNil(t, notification.CreatedAt)
	assert.NotNil(t, notification.UpdatedAt)
}

func TestService_processPending_success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	svc := newTestService(pool)
	ctx := context.Background()

	req := NotificationInput{
		Provider:  string(db.NotifierProviderTELEGRAM),
		Recipient: "12345678",
		Title:     "Test Alert",
		Body:      "This is a test notification",
	}
	result, err := sendTestNotification(ctx, svc, req)
	require.NoError(t, err)

	processed, err := svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	notification, err := getTestNotification(ctx, svc, result.NotificationID)
	require.NoError(t, err)
	assert.Equal(t, db.NotifierNotificationStatusSENT, notification.Status)
	assert.Equal(t, int32(0), notification.RetryCount)
}

func TestService_processPending_failureAndRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	svc := newTestService(pool)
	ctx := context.Background()

	req := NotificationInput{
		Provider:  string(db.NotifierProviderTELEGRAM),
		Recipient: "12345678",
		Title:     "Test Alert",
		Body:      "trigger_failure",
	}
	result, err := sendTestNotification(ctx, svc, req)
	require.NoError(t, err)

	processed, err := svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	notification, err := getTestNotification(ctx, svc, result.NotificationID)
	require.NoError(t, err)
	assert.Equal(t, db.NotifierNotificationStatusPENDING, notification.Status)
	assert.Equal(t, int32(1), notification.RetryCount)
	assert.Contains(t, notification.ErrorMessage, "send failure triggered")
}

func TestService_processPending_permanentFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	breakers := NewBreakers(10, 2, 10*time.Second)
	svc := newTestServiceWithBreakers(pool, breakers)
	ctx := context.Background()

	req := NotificationInput{
		Provider:  string(db.NotifierProviderTELEGRAM),
		Recipient: "12345678",
		Title:     "Test Alert",
		Body:      "trigger_failure",
	}
	result, err := sendTestNotification(ctx, svc, req)
	require.NoError(t, err)

	id, err := uuid.Parse(result.NotificationID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "UPDATE notify.notifications SET retry_count = 4, updated_at = now() - interval '60 seconds' WHERE id = $1", pgtype.UUID{Bytes: id, Valid: true})
	require.NoError(t, err)

	processed, err := svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	notification, err := getTestNotification(ctx, svc, result.NotificationID)
	require.NoError(t, err)
	assert.Equal(t, db.NotifierNotificationStatusFAILED, notification.Status)
	assert.Equal(t, int32(maxDeliveryAttempts), notification.RetryCount)
}

func TestService_processPending_circuitBreaker(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	breaker := NewCircuitBreaker(2, 2, 10*time.Second)
	breakers := NewBreakers(2, 2, 10*time.Second)
	breakers.Telegram = breaker
	svc := newTestServiceWithBreakers(pool, breakers)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := sendTestNotification(ctx, svc, NotificationInput{
			Provider:  string(db.NotifierProviderTELEGRAM),
			Recipient: "12345678",
			Title:     fmt.Sprintf("Test Alert %d", i),
			Body:      "trigger_failure",
		})
		require.NoError(t, err)
	}

	processed, err := svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 3, processed)
	assert.Equal(t, CircuitOpen, breaker.State())

	rows, err := pool.Query(ctx, "SELECT error_message FROM notify.notifications WHERE error_message IS NOT NULL")
	require.NoError(t, err)
	defer rows.Close()

	foundBreakerErr := false
	for rows.Next() {
		var errMsg string
		err = rows.Scan(&errMsg)
		require.NoError(t, err)
		if strings.Contains(errMsg, "circuit breaker is open") {
			foundBreakerErr = true
		}
	}
	assert.True(t, foundBreakerErr, "expected at least one notification to fail with circuit breaker open error")
}

func TestService_processPending_exponentialBackoff(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	svc := newTestService(pool)
	ctx := context.Background()

	req := NotificationInput{
		Provider:  string(db.NotifierProviderTELEGRAM),
		Recipient: "12345678",
		Title:     "Test Alert",
		Body:      "trigger_failure",
	}
	result, err := sendTestNotification(ctx, svc, req)
	require.NoError(t, err)

	processed, err := svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	processed, err = svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 0, processed, "expected notification to be skipped due to exponential backoff")

	id, err := uuid.Parse(result.NotificationID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "UPDATE notify.notifications SET updated_at = now() - interval '10 seconds' WHERE id = $1", pgtype.UUID{Bytes: id, Valid: true})
	require.NoError(t, err)

	processed, err = svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 1, processed, "expected notification to be processed after backoff duration elapsed")
}

func TestService_processPending_deduplication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	svc := newTestService(pool)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := sendTestNotification(ctx, svc, NotificationInput{
			Provider:  string(db.NotifierProviderTELEGRAM),
			Recipient: "12345678",
			Title:     "Deduplicated Alert",
			Body:      fmt.Sprintf("Alert details for node %d", i),
		})
		require.NoError(t, err)
	}

	processed, err := svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 5, processed, "expected all 5 notifications to be processed as part of deduplication group")

	sentCount, err := countNotificationsByStatus(ctx, pool, db.NotifierNotificationStatusSENT)
	require.NoError(t, err)
	assert.Equal(t, 5, sentCount)

	faultproof.Log(t, "notifier_deduplication", map[string]string{
		"size":    "5",
		"success": "true",
	})
}

func TestService_processPending_fallback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	breakers := NewBreakers(3, 2, 10*time.Second)
	breakers.Slack.trip()
	svc := newTestServiceWithBreakers(pool, breakers)
	ctx := context.Background()

	req := NotificationInput{
		Provider:  string(db.NotifierProviderSLACK),
		Recipient: "https://hooks.slack.com/services/test",
		Title:     "Fallback Alert",
		Body:      "This notification falls back",
	}

	result, err := sendTestNotification(ctx, svc, req)
	require.NoError(t, err)

	processed, err := svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	notification, err := getTestNotification(ctx, svc, result.NotificationID)
	require.NoError(t, err)
	assert.Equal(t, db.NotifierNotificationStatusSENT, notification.Status)
	assert.Equal(t, string(db.NotifierProviderTELEGRAM), notification.Provider)

	faultproof.Log(t, "notifier_fallback", map[string]string{
		"primary":  "SLACK",
		"fallback": "TELEGRAM",
		"success":  "true",
	})
}

func TestService_processPending_broadcast(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	cfg, breakers := newBroadcastTestConfig("")
	svc := NewService(pool, cfg, breakers)
	ctx := context.Background()

	result, err := sendTestNotification(ctx, svc, NotificationInput{
		Provider:  string(db.NotifierProviderTELEGRAM),
		Recipient: "12345678",
		Title:     "Broadcast Alert",
		Body:      "fan-out to all channels",
		Broadcast: true,
		BroadcastProviders: []string{
			string(db.NotifierProviderSLACK),
			string(db.NotifierProviderTELEGRAM),
		},
	})
	require.NoError(t, err)

	processed, err := svc.ProcessPending(ctx, workerBatchSize)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	notification, err := getTestNotification(ctx, svc, result.NotificationID)
	require.NoError(t, err)
	assert.Equal(t, db.NotifierNotificationStatusSENT, notification.Status)
	assert.Equal(t, db.NotifierDeliveryModeBROADCAST, notification.DeliveryMode)
	assert.Equal(t, string(db.NotifierProviderTELEGRAM), notification.Provider)
}

func TestSend_interactiveButtons(t *testing.T) {
	ctx := context.WithValue(context.Background(), NotificationIDContextKey, "test-notification-uuid-123")
	actions := BuildInteractiveActions("test-notification-uuid-123", "Critical Error", "Intruder detected on IP 192.168.1.100")

	telegramMsg := telegramPayload{
		ChatID:    "chat-id",
		Text:      "<b>Critical Error</b>\n\nIntruder detected on IP 192.168.1.100",
		ParseMode: "HTML",
	}
	var telegramRows []telegramButtonRow
	if actions.AcknowledgeURL != "" {
		telegramRows = append(telegramRows, telegramButtonRow{{Text: "✅ Acknowledge Incident", URL: actions.AcknowledgeURL}})
	}
	if actions.BlockIPURL != "" {
		telegramRows = append(telegramRows, telegramButtonRow{{Text: "🚫 Block IP " + actions.BlockIP, URL: actions.BlockIPURL}})
	}
	telegramMsg.ReplyMarkup = &telegramReplyMarkup{InlineKeyboard: telegramRows}

	telegramBytes, err := json.Marshal(telegramMsg)
	require.NoError(t, err)

	var telegramDecoded telegramPayload
	require.NoError(t, json.Unmarshal(telegramBytes, &telegramDecoded))
	assert.Equal(t, "chat-id", telegramDecoded.ChatID)
	assert.Equal(t, "HTML", telegramDecoded.ParseMode)
	require.NotNil(t, telegramDecoded.ReplyMarkup)
	assert.Len(t, telegramDecoded.ReplyMarkup.InlineKeyboard, 2)

	text := "*Critical Error*\nIntruder detected on IP 10.0.0.5"
	blocks := []slackBlock{
		{Type: "section", Text: &slackText{Type: "mrkdwn", Text: text}},
	}
	var buttons []slackButton
	if actions.AcknowledgeURL != "" {
		buttons = append(buttons, slackButton{Type: "button", Text: slackText{Type: "plain_text", Text: "✅ Acknowledge"}, URL: actions.AcknowledgeURL})
	}
	if actions.BlockIPURL != "" {
		buttons = append(buttons, slackButton{Type: "button", Style: "danger", Text: slackText{Type: "plain_text", Text: "🚫 Block IP " + actions.BlockIP}, URL: actions.BlockIPURL})
	}
	blocks = append(blocks, slackBlock{Type: "actions", Elements: buttons})

	slackBytes, err := json.Marshal(slackBlocksPayload{Blocks: blocks})
	require.NoError(t, err)

	var slackDecoded slackBlocksPayload
	require.NoError(t, json.Unmarshal(slackBytes, &slackDecoded))
	assert.Len(t, slackDecoded.Blocks, 2)

	_ = ctx

	faultproof.Log(t, "notifier_interactive_buttons", map[string]string{"success": "true"})
}
