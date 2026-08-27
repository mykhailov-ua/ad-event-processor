package ingestion

import (
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestTryAcquireStreamAdmission_holdoutDeferredNoPublisher(t *testing.T) {
	cfg := &config.Config{StreamProducerAdmissionPct: 85}
	sharder := NewJumpHashSharder(1)
	campaignID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	_, kind, acquired := tryAcquireStreamAdmission(cfg, sharder, nil, nil, campaignID, true)
	require.False(t, acquired, "deferred ingest must fail closed without broker or stream producer")
	require.Equal(t, filterRejectInfra, kind)
}

func TestPublishAcceptedTrack_holdoutDeferredNoPublisher(t *testing.T) {
	redisClient := redis.NewUniversalClient(&redis.UniversalOptions{Addrs: []string{"127.0.0.1:1"}})
	defer func() { _ = redisClient.Close() }()

	uf := NewUnifiedFilter([]redis.UniversalClient{redisClient}, nil, nil, nil, 0, time.Minute, time.Hour, time.Hour, 100, 10, "events", 1000)
	uf.SetDeferStreamToProducer(true)
	engine := NewFilterEngine(0, uf)

	cfg := &config.Config{}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, engine, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)

	evt := &domain.Event{
		CampaignID: uuid.New(),
		ClickID:    "holdout-deferred-no-sink",
		Type:       "click",
	}

	require.False(t, h.publishAcceptedTrack(evt, nil), "deferred mode must not no-op accept without publisher")
}

func TestPublishAcceptedTrack_holdoutAllowsInlineLuaWithoutPublisher(t *testing.T) {
	redisClient := redis.NewUniversalClient(&redis.UniversalOptions{Addrs: []string{"127.0.0.1:1"}})
	defer func() { _ = redisClient.Close() }()

	uf := NewUnifiedFilter([]redis.UniversalClient{redisClient}, nil, nil, nil, 0, time.Minute, time.Hour, time.Hour, 100, 10, "events", 1000)
	require.False(t, uf.StreamDeferredToProducer())
	engine := NewFilterEngine(0, uf)

	cfg := &config.Config{}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, engine, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)

	evt := &domain.Event{
		CampaignID: uuid.New(),
		ClickID:    "holdout-inline-lua",
		Type:       "click",
	}

	require.True(t, h.publishAcceptedTrack(evt, nil), "non-deferred Lua path may no-op publish when stream write stays in Lua")
}

func TestTrackIngestPublisherReady_brokerOrStream(t *testing.T) {
	campaignID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	sharder := NewJumpHashSharder(1)

	require.False(t, trackIngestPublisherReady(sharder, nil, nil, campaignID))

	p := NewStreamProducer(redis.NewUniversalClient(&redis.UniversalOptions{Addrs: []string{"127.0.0.1:1"}}), "events", 8, 0)
	defer p.Close()
	require.True(t, trackIngestPublisherReady(sharder, []*StreamProducer{p}, nil, campaignID))
}
