package scorer

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/fraud/features"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	redis "github.com/redis/go-redis/v9"
)

type aggKey struct {
	IP         string
	CampaignID string
}

type aggStats struct {
	Events      uint64
	Clicks      uint64
	UniqueUsers map[string]struct{}
	UniqueUAs   map[string]struct{}
}

type MicroBatcher struct {
	eventsChan            chan *domain.Event
	redisShards           []redis.UniversalClient
	scorer                Scorer
	campaignUpdateChannel string
	cfg                   MicroBatcherConfig
	paused                atomic.Bool
	lastBoosts            atomic.Value
	BoostScore            func(row features.FeatureRow, mlProbability float64) (int, bool)
}

const ScoreBoostTTL = 900 * time.Second

func defaultBoostScore(row features.FeatureRow, mlProbability float64) (int, bool) {
	tier, score := MapProbabilityTier(mlProbability)
	if tier != FraudTierSuspect {
		return 0, false
	}
	_ = row
	return score, true
}

func NewMicroBatcher(redisShards []redis.UniversalClient, scorer Scorer, campaignUpdateChannel string, cfg MicroBatcherConfig) *MicroBatcher {
	return &MicroBatcher{
		eventsChan:            make(chan *domain.Event, 10000),
		redisShards:           redisShards,
		scorer:                scorer,
		campaignUpdateChannel: domain.DefaultCampaignUpdateChannel(campaignUpdateChannel),
		cfg:                   cfg.normalized(),
	}
}

func (m *MicroBatcher) Enqueue(evt *domain.Event, msgID string) {
	if evt == nil {
		return
	}

	parts := strings.Split(msgID, "-")
	if len(parts) > 0 {
		ms, err := strconv.ParseInt(parts[0], 10, 64)
		if err == nil {
			nowMs := time.Now().UnixNano() / int64(time.Millisecond)
			lagSec := float64(nowMs-ms) / 1000.0
			if lagSec < 0 {
				lagSec = 0
			}
			metrics.ProcessorStreamLagSeconds.WithLabelValues("fraud").Set(lagSec)

			if lagSec > m.cfg.MaxStreamLagSec {
				m.paused.Store(true)
				metrics.MicroBatchPaused.Set(1)
				return
			}
		}
	}
	m.paused.Store(false)
	metrics.MicroBatchPaused.Set(0)

	select {
	case m.eventsChan <- evt:
		metrics.MicroBatchProcessedTotal.Inc()
	default:
		metrics.MicroBatchDroppedTotal.Inc()
	}
}

func (m *MicroBatcher) Start(ctx context.Context) {
	ticker := time.NewTicker(m.cfg.FlushInterval)
	refreshTicker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	defer refreshTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.flush(ctx)
		case <-refreshTicker.C:
			if m.paused.Load() {
				m.refreshBoostTTL(ctx)
			}
		}
	}
}

func (m *MicroBatcher) flush(ctx context.Context) {
	if m.scorer == nil || len(m.redisShards) == 0 {
		return
	}

	batch := make(map[aggKey]*aggStats)
	limit := 10000

	for i := 0; i < limit; i++ {
		select {
		case evt := <-m.eventsChan:
			key := aggKey{IP: evt.IP, CampaignID: evt.CampaignID.String()}
			stats, ok := batch[key]
			if !ok {
				stats = &aggStats{
					UniqueUsers: make(map[string]struct{}),
					UniqueUAs:   make(map[string]struct{}),
				}
				batch[key] = stats
			}
			stats.Events++
			if evt.Type == "click" {
				stats.Clicks++
			}
			if evt.UserID != "" {
				stats.UniqueUsers[evt.UserID] = struct{}{}
			}
			if evt.UA != "" {
				stats.UniqueUAs[evt.UA] = struct{}{}
			}
		default:
			i = limit
		}
	}

	if len(batch) == 0 {
		return
	}

	rows := make([]features.FeatureRow, 0, len(batch))
	keys := make([]aggKey, 0, len(batch))
	for key, stats := range batch {
		rows = append(rows, features.FeatureRow{
			WindowStart:      time.Now(),
			IPAddress:        key.IP,
			CampaignID:       key.CampaignID,
			Events:           stats.Events,
			Clicks:           stats.Clicks,
			SpendMicro:       0,
			BudgetLimitMicro: 0,
			UniqueUsers:      uint64(len(stats.UniqueUsers)),
			UniqueUAs:        uint64(len(stats.UniqueUAs)),
		})
		keys = append(keys, key)
	}

	scores, err := m.scorer.ScoreBatch(ctx, rows)
	if err != nil {
		slog.Error("micro-batch scorer prediction failed", "error", err)
		return
	}

	campaignBoost := make(map[string]int, len(rows))
	boostFn := m.BoostScore
	if boostFn == nil {
		boostFn = defaultBoostScore
	}
	for i, score := range scores {
		boost, ok := boostFn(rows[i], score)
		if !ok {
			continue
		}
		campaignID := keys[i].CampaignID
		if prev, exists := campaignBoost[campaignID]; !exists || boost > prev {
			campaignBoost[campaignID] = boost
		}
	}

	for campaignID, fraudScore := range campaignBoost {
		key := fmt.Sprintf("ml:score:boost:%s", campaignID)
		value := strconv.Itoa(fraudScore)
		if err := database.SyncGlobalStringToAllShards(ctx, m.redisShards, key, value, ScoreBoostTTL); err != nil {
			slog.Error("failed to set micro-batch ml score boost to redis", "error", err, "campaign", campaignID)
			continue
		}
		if err := domain.PublishCampaignUpdateRedis(ctx, m.redisShards, m.campaignUpdateChannel, campaignID); err != nil {
			slog.Warn("failed to publish campaign update after ml boost", "error", err, "campaign", campaignID)
		}
		metrics.MicroBatchBoostsWrittenTotal.Inc()
	}
	if len(campaignBoost) > 0 {
		copyBoosts := make(map[string]int, len(campaignBoost))
		for campaignID, fraudScore := range campaignBoost {
			copyBoosts[campaignID] = fraudScore
		}
		m.lastBoosts.Store(copyBoosts)
	}
}

func (m *MicroBatcher) refreshBoostTTL(ctx context.Context) {
	if m == nil || len(m.redisShards) == 0 {
		return
	}
	raw := m.lastBoosts.Load()
	if raw == nil {
		return
	}
	boosts, ok := raw.(map[string]int)
	if !ok || len(boosts) == 0 {
		return
	}
	for campaignID, fraudScore := range boosts {
		key := fmt.Sprintf("ml:score:boost:%s", campaignID)
		value := strconv.Itoa(fraudScore)
		if err := database.SyncGlobalStringToAllShards(ctx, m.redisShards, key, value, ScoreBoostTTL); err != nil {
			slog.Warn("failed to refresh micro-batch ml score boost ttl", "error", err, "campaign", campaignID)
		}
	}
}
