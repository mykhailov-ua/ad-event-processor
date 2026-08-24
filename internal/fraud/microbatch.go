package fraud

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

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
	redisShards                  []redis.UniversalClient
	scorer                Scorer
	campaignUpdateChannel string
}

func NewMicroBatcher(redisShards []redis.UniversalClient, scorer Scorer, campaignUpdateChannel string) *MicroBatcher {
	return &MicroBatcher{
		eventsChan:            make(chan *domain.Event, 10000),
		redisShards:                  redisShards,
		scorer:                scorer,
		campaignUpdateChannel: domain.DefaultCampaignUpdateChannel(campaignUpdateChannel),
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

			if lagSec > 30 {
				metrics.MicroBatchPaused.Set(1)
				return
			}
		}
	}
	metrics.MicroBatchPaused.Set(0)

	select {
	case m.eventsChan <- evt:
		metrics.MicroBatchProcessedTotal.Inc()
	default:
	}
}

func (m *MicroBatcher) Start(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.flush(ctx)
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

	rows := make([]FeatureRow, 0, len(batch))
	keys := make([]aggKey, 0, len(batch))
	for key, stats := range batch {
		rows = append(rows, FeatureRow{
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
	for i, score := range scores {
		boost, ok := microbatchBoostScore(rows[i], score)
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
}
