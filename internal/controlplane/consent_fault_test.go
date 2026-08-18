package controlplane

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func signConsentBody(secret []byte, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestFault_ConsentWebhookReplay(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}
	secret := []byte("consent-test-secret")
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{ConsentHMACSecret: config.Secret(secret)}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	h := NewHandler(svc, cfg, nil, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, _ := json.Marshal(ConsentRecord{
		UserID:   "replay-user",
		Purposes: domain.ConsentPurposeAdStorage,
		Source:   "cmp",
	})
	sig := signConsentBody(secret, body)

	for i := range 2 {
		req, _ := http.NewRequest("POST", "/api/v1/consent", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Consent-Signature", sig)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		require.Equal(t, http.StatusNoContent, rr.Code, "attempt %d", i+1)
	}

	var count int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM consent_events`).Scan(&count))
	assert.Equal(t, 2, count)
}

func TestFault_ConsentReadYourWrites(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}
	ctx := context.Background()
	secret := []byte("consent-ryw-secret")
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		ConsentHMACSecret:    config.Secret(secret),
		ConsentUpdateChannel: "test:consent:update",
	}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	store := domain.NewConsentStore(rdb)
	store.StartWatch(ctx, rdb, cfg.ConsentUpdateChannel)
	worker := NewOutboxWorker(svc)

	require.NoError(t, svc.RecordConsent(ctx, ConsentRecord{
		UserID:   "ryw-user",
		Purposes: domain.ConsentPurposeAdStorage | domain.ConsentPurposeAnalytics,
		Source:   "web",
	}))
	require.NoError(t, worker.ProcessOutbox(ctx))

	want := domain.ConsentPurposeAdStorage | domain.ConsentPurposeAnalytics
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if store.PurposesForUser("ryw-user") == want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("fault_proof fault=consent_read_your_writes consent not visible within 2s")
}

func TestFault_ErasurePartialShardFailure(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}
	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	okRdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()
	badRdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})

	cfg := &config.Config{ConsentUpdateChannel: "test:erasure"}
	svc := newBareService(t, pool, []redis.UniversalClient{okRdb, badRdb}, cfg)

	userID := "erasure-user"
	hashHex := domain.HashUserIDHex(userID)
	require.NoError(t, okRdb.Set(ctx, domain.ConsentRedisKeyPrefix+hashHex, "3", 0).Err())

	reqID, err := svc.CreatePrivacyErasureRequest(ctx, userID)
	require.NoError(t, err)
	require.NoError(t, svc.ProcessPrivacyErasureTick(ctx))
	require.NoError(t, NewOutboxWorker(svc).ProcessOutbox(ctx))

	var status string
	require.NoError(t, pool.QueryRow(ctx, `SELECT status::text FROM privacy_erasure_requests WHERE id = $1`, domain.ToUUID(reqID)).Scan(&status))
	assert.Equal(t, "REDIS_PURGED", status)
}
