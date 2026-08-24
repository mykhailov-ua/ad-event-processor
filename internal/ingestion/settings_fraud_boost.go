package ingestion

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	fraudScoreBoostKeyPrefix = "ml:score:boost:"
	fraudBoostFullResync     = 30 * time.Second
)

func fraudScoreBoostKey(campaignID uuid.UUID) string {
	return fraudScoreBoostKeyPrefix + campaignID.String()
}

func parseFraudBoostCampaignID(key string) (uuid.UUID, bool) {
	if !strings.HasPrefix(key, fraudScoreBoostKeyPrefix) {
		return uuid.UUID{}, false
	}
	campIDStr := key[len(fraudScoreBoostKeyPrefix):]
	var campID uuid.UUID
	if !ParseUUID(UnsafeBytes(campIDStr), &campID) {
		return uuid.UUID{}, false
	}
	return campID, true
}

func parseCampaignUpdateCampaignID(payload string) (uuid.UUID, bool) {
	if payload == "" || domain.IsRegistryFullSyncPayload(payload) {
		return uuid.UUID{}, false
	}
	id, err := uuid.Parse(payload)
	if err != nil {
		return uuid.UUID{}, false
	}
	return id, true
}

func clampFraudBoostScore(val int) uint8 {
	if val < 0 {
		return 0
	}
	if val > 100 {
		return 100
	}
	return uint8(val)
}

func parseFraudBoostValue(raw string) (uint8, bool) {
	val, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return clampFraudBoostScore(val), true
}

func cloneFraudBoostMap(src map[uuid.UUID]uint8) map[uuid.UUID]uint8 {
	if len(src) == 0 {
		return make(map[uuid.UUID]uint8)
	}
	dst := make(map[uuid.UUID]uint8, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func loadAllFraudBoostsFromShard(ctx context.Context, rdb redis.UniversalClient) (map[uuid.UUID]uint8, error) {
	newBoosts := make(map[uuid.UUID]uint8)
	cursor := uint64(0)
	for {
		keys, next, err := rdb.Scan(ctx, cursor, fraudScoreBoostKeyPrefix+"*", 100).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			campID, ok := parseFraudBoostCampaignID(key)
			if !ok {
				continue
			}
			valStr, err := rdb.Get(ctx, key).Result()
			if err != nil {
				continue
			}
			score, ok := parseFraudBoostValue(valStr)
			if !ok {
				continue
			}
			newBoosts[campID] = score
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return newBoosts, nil
}

func (sw *SettingsWatcher) syncFraudScoreBoostsFull(ctx context.Context) {
	rdb := sw.pickHealthyShard()
	if rdb == nil {
		return
	}

	for attempt := 0; attempt < len(sw.rdbs); attempt++ {
		newBoosts, err := loadAllFraudBoostsFromShard(ctx, rdb)
		if err != nil {
			slog.Warn("failed to scan ml boost keys from redis, trying next shard", "error", err)
			rdb = sw.nextShardAfter(rdb)
			if rdb == nil {
				return
			}
			continue
		}
		sw.fraudScoreBoosts.Store(&FraudScoreBoostSnapshot{Boosts: newBoosts})
		return
	}
}

func (sw *SettingsWatcher) applyFraudBoostCampaign(ctx context.Context, campaignID uuid.UUID) {
	rdb := sw.pickHealthyShard()
	if rdb == nil {
		return
	}

	prev := sw.GetFraudScoreBoosts()
	next := cloneFraudBoostMap(prev.Boosts)

	valStr, err := rdb.Get(ctx, fraudScoreBoostKey(campaignID)).Result()
	if errors.Is(err, redis.Nil) {
		delete(next, campaignID)
		sw.fraudScoreBoosts.Store(&FraudScoreBoostSnapshot{Boosts: next})
		return
	}
	if err != nil {
		slog.Warn("failed to read ml boost key", "campaign_id", campaignID, "error", err)
		return
	}
	score, ok := parseFraudBoostValue(valStr)
	if !ok {
		delete(next, campaignID)
	} else {
		next[campaignID] = score
	}
	sw.fraudScoreBoosts.Store(&FraudScoreBoostSnapshot{Boosts: next})
}

func (sw *SettingsWatcher) runFraudBoostSubscriber(ctx context.Context) {
	channel := domain.DefaultCampaignUpdateChannel(sw.campaignUpdateChannel)
	backoff := time.Second

	for ctx.Err() == nil {
		if !sw.consumeFraudBoostUpdates(ctx, channel) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (sw *SettingsWatcher) consumeFraudBoostUpdates(ctx context.Context, channel string) bool {
	rdb := sw.pickHealthyShard()
	if rdb == nil {
		return true
	}

	pubsub := rdb.Subscribe(ctx, channel)
	defer func() { _ = pubsub.Close() }()

	for msg := range pubsub.Channel() {
		if ctx.Err() != nil {
			return false
		}
		if domain.IsRegistryFullSyncPayload(msg.Payload) {
			sw.syncFraudScoreBoostsFull(ctx)
			continue
		}
		campID, ok := parseCampaignUpdateCampaignID(msg.Payload)
		if !ok {
			continue
		}
		sw.applyFraudBoostCampaign(ctx, campID)
	}
	return true
}
