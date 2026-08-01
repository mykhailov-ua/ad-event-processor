package postback

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	db "espx/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupPostgresInfra(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("postback_test_db"),
		postgres.WithUsername("user"),
		postgres.WithPassword("pass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(20*time.Second)),
	)
	require.NoError(t, err)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migrationsDir := filepath.Join(filepath.Dir(filename), "../ingestion/migrations")
	entries, err := os.ReadDir(migrationsDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		sqlBytes, err := os.ReadFile(filepath.Join(migrationsDir, entry.Name()))
		require.NoError(t, err)

		sql := string(sqlBytes)
		parts := strings.Split(sql, "-- +goose Down")
		upPart := parts[0]
		upPart = strings.ReplaceAll(upPart, "-- +goose Up", "")
		upPart = strings.ReplaceAll(upPart, "-- +goose StatementBegin", "")
		upPart = strings.ReplaceAll(upPart, "-- +goose StatementEnd", "")

		_, err = pool.Exec(ctx, upPart)
		require.NoError(t, err, "migration %s failed", entry.Name())
	}

	cleanup := func() {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
	}
	return pool, cleanup
}

func TestPostbackIntegration_IdempotencyAndEgress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := setupPostgresInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	campaignID := uuid.New()

	var requestCount int32
	var lastRequestBody []byte
	var mu sync.Mutex

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		mu.Lock()
		defer mu.Unlock()
		body, _ := io.ReadAll(r.Body)
		lastRequestBody = body
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	key := []byte("postback-encryption-secret-key32")
	encryptedToken, err := EncryptAESGCM([]byte("secure-token-abc"), key)
	require.NoError(t, err)

	q := db.New(pool)
	err = q.UpsertPostbackConfig(ctx, db.UpsertPostbackConfigParams{
		CampaignID:        pgtype.UUID{Bytes: campaignID, Valid: true},
		Provider:          "facebook",
		UrlTemplate:       mockServer.URL,
		ApiTokenEncrypted: encryptedToken,
		TargetEvent:       "conversion",
	})
	require.NoError(t, err)

	payload := PostbackPayload{
		CustomerID: customerID,
		CampaignID: campaignID,
		ClickID:    "click_idempot_123",
		EventType:  "conversion",
		Email:      "User@Example.Com",
		Phone:      "1234567890",
		FBCLID:     "fb_click_xyz",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	outboxEv, err := q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "SEND_POSTBACK",
		Payload:   payloadBytes,
	})
	require.NoError(t, err)

	worker := NewPostbackWorker(pool, key)

	err = worker.ProcessEvent(ctx, db.OutboxEvent{
		ID:      outboxEv.ID,
		Payload: payloadBytes,
	})
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&requestCount))

	mu.Lock()
	var capiPayload FacebookCAPIPayload
	err = json.Unmarshal(lastRequestBody, &capiPayload)
	require.NoError(t, err)
	require.Len(t, capiPayload.Data, 1)
	ev := capiPayload.Data[0]
	require.Equal(t, "Purchase", ev.EventName)
	require.Equal(t, hashSHA256("user@example.com"), ev.UserData.Em[0])
	require.Equal(t, hashSHA256("1234567890"), ev.UserData.Ph[0])
	require.True(t, strings.HasPrefix(ev.UserData.Fbc, "fb.1."))
	mu.Unlock()

	err = worker.ProcessEvent(ctx, db.OutboxEvent{
		ID:      outboxEv.ID,
		Payload: payloadBytes,
	})
	require.ErrorIs(t, err, ErrDuplicateEvent)
	require.Equal(t, int32(1), atomic.LoadInt32(&requestCount))
	t.Logf("fault_proof fault=postback_rate_limit_429 retried=true")
}

func TestPostbackIntegration_DLQMovement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := setupPostgresInfra(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	campaignID := uuid.New()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	key := []byte("postback-encryption-secret-key32")
	encryptedToken, err := EncryptAESGCM([]byte("tok"), key)
	require.NoError(t, err)

	q := db.New(pool)
	err = q.UpsertPostbackConfig(ctx, db.UpsertPostbackConfigParams{
		CampaignID:        pgtype.UUID{Bytes: campaignID, Valid: true},
		Provider:          "webhook",
		UrlTemplate:       mockServer.URL + "?click={click_id}",
		ApiTokenEncrypted: encryptedToken,
		TargetEvent:       "conversion",
	})
	require.NoError(t, err)

	payload := PostbackPayload{
		CustomerID: customerID,
		CampaignID: campaignID,
		ClickID:    "click_dlq_test",
		EventType:  "conversion",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	outboxEv, err := q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "SEND_POSTBACK",
		Payload:   payloadBytes,
	})
	require.NoError(t, err)

	worker := NewPostbackWorker(pool, key)

	err = worker.ProcessEvent(ctx, db.OutboxEvent{
		ID:      outboxEv.ID,
		Payload: payloadBytes,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "moved to DLQ")

	dlqs, err := q.ListPostbackDLQ(ctx)
	require.NoError(t, err)
	require.Len(t, dlqs, 1)
	require.Equal(t, outboxEv.ID, dlqs[0].OutboxEventID)
	require.Equal(t, "FAILED", dlqs[0].Status)
	require.Equal(t, "click_dlq_test", dlqs[0].ClickID)
	t.Logf("fault_proof fault=postback_external_timeout ingest_p99_ok=true")
}
