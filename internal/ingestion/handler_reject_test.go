package ingestion

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

type mockRedisXAdd struct {
	mockRedisClient
}

func (m *mockRedisXAdd) XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	cmd.SetVal("1-0")
	return cmd
}

func TestAdsPacketHandler_FilterErrors(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize: 1024 * 1024,
		FilterTimeoutMs:    50,
		StreamMaxLen:       1000,
	}
	sharder := NewJumpHashSharder(1)
	registry := &mockRegistry{}

	makeBody := func() []byte {
		return []byte(`{"campaign_id":"` + uuid.NewString() + `","type":"click","click_id":"c1"}`)
	}

	t.Run("ErrCampaignNotFound -> 404", func(t *testing.T) {
		h := NewAdsPacketHandler(cfg, registry, NewFilterEngine(0, &errFilter{err: ErrCampaignNotFound}), nil, nil, sharder, "fraud", nil)
		status, written := PostTrackGnetJSON(h, makeBody())
		assert.Equal(t, http.StatusNotFound, status)
		assert.True(t, bytes.HasPrefix(written, []byte("HTTP/1.1 404")))
	})

	t.Run("ErrBidFloorNotMet -> 402", func(t *testing.T) {
		h := NewAdsPacketHandler(cfg, registry, NewFilterEngine(0, &errFilter{err: ErrBidFloorNotMet}), nil, nil, sharder, "fraud", nil)
		status, written := PostTrackGnetJSON(h, makeBody())
		assert.Equal(t, http.StatusPaymentRequired, status)
		assert.True(t, bytes.HasPrefix(written, []byte("HTTP/1.1 402")))
	})

	t.Run("filter timeout -> 504", func(t *testing.T) {
		h := NewAdsPacketHandler(cfg, registry, NewFilterEngine(50*time.Millisecond, &slowFilter{delay: 200 * time.Millisecond}), nil, nil, sharder, "fraud", nil)
		status, written := PostTrackGnetJSON(h, makeBody())
		assert.Equal(t, http.StatusGatewayTimeout, status)
		assert.True(t, bytes.HasPrefix(written, []byte("HTTP/1.1 504")))
	})

	t.Run("ErrFraudDetected silent reject on -> 202", func(t *testing.T) {
		configureMockRegistryCampaign(func(c *domain.Campaign) {
			c.SilentRejectEnabled = true
		})
		redisClient := &mockRedisXAdd{}
		h := NewAdsPacketHandler(cfg, registry, NewFilterEngine(0, &errFilter{err: ErrFraudDetected}), nil, []redis.UniversalClient{redisClient}, sharder, "fraud-stream", nil)
		status, written := PostTrackGnetJSON(h, makeBody())
		assert.Equal(t, http.StatusAccepted, status)
		assert.True(t, bytes.HasPrefix(written, []byte("HTTP/1.1 202")))
	})

	t.Run("ErrFraudDetected silent reject off -> 403 holdout", func(t *testing.T) {
		configureMockRegistryCampaign(func(c *domain.Campaign) {
			c.SilentRejectEnabled = false
		})
		redisClient := &mockRedisXAdd{}
		h := NewAdsPacketHandler(cfg, registry, NewFilterEngine(0, &errFilter{err: ErrFraudDetected}), nil, []redis.UniversalClient{redisClient}, sharder, "fraud-stream", nil)
		status, written := PostTrackGnetJSON(h, makeBody())
		assert.Equal(t, http.StatusForbidden, status)
		assert.True(t, bytes.HasPrefix(written, []byte("HTTP/1.1 403")))
	})

	t.Run("redis circuit open -> 503 Retry-After", func(t *testing.T) {
		h := NewAdsPacketHandler(cfg, registry, NewFilterEngine(0, &errFilter{err: database.ErrRedisCircuitOpen}), nil, nil, sharder, "fraud", nil)
		status, written := PostTrackGnetJSON(h, makeBody())
		assert.Equal(t, http.StatusServiceUnavailable, status)
		assert.True(t, bytes.HasPrefix(written, []byte("HTTP/1.1 503")))
		assert.True(t, bytes.Contains(written, []byte("Retry-After: 1")))
	})

	t.Run("ErrRateLimitExceeded -> 429 Retry-After", func(t *testing.T) {
		h := NewAdsPacketHandler(cfg, registry, NewFilterEngine(0, &errFilter{err: ErrRateLimitExceeded}), nil, nil, sharder, "fraud", nil)
		status, written := PostTrackGnetJSON(h, makeBody())
		assert.Equal(t, http.StatusTooManyRequests, status)
		assert.True(t, bytes.Contains(written, []byte("Retry-After: 60")))
	})
}
