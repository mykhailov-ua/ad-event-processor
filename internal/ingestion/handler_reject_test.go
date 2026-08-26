package ingestion

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		redisClient := &countingRedisXAdd{}
		h := NewAdsPacketHandler(cfg, registry, NewFilterEngine(0, &errFilter{err: ErrFraudDetected}), nil, []redis.UniversalClient{redisClient}, sharder, "fraud-stream", nil)
		status, written := PostTrackGnetJSON(h, makeBody())
		assert.Equal(t, http.StatusAccepted, status)
		assert.True(t, bytes.HasPrefix(written, []byte("HTTP/1.1 202")))
		require.Eventually(t, func() bool {
			return redisClient.xadds.Load() >= 1
		}, time.Second, 2*time.Millisecond)
	})

	t.Run("ErrFraudDetected silent reject off -> 403 holdout", func(t *testing.T) {
		configureMockRegistryCampaign(func(c *domain.Campaign) {
			c.SilentRejectEnabled = false
		})
		redisClient := &countingRedisXAdd{}
		h := NewAdsPacketHandler(cfg, registry, NewFilterEngine(0, &errFilter{err: ErrFraudDetected}), nil, []redis.UniversalClient{redisClient}, sharder, "fraud-stream", nil)
		status, written := PostTrackGnetJSON(h, makeBody())
		assert.Equal(t, http.StatusForbidden, status)
		assert.True(t, bytes.HasPrefix(written, []byte("HTTP/1.1 403")))
		require.Eventually(t, func() bool {
			return redisClient.xadds.Load() >= 1
		}, time.Second, 2*time.Millisecond)
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
