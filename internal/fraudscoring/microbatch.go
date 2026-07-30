package fraudscoring

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"espx/internal/campaignmodel"
	"espx/internal/metrics"

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
	eventsChan chan *campaignmodel.Event
	rdb        redis.UniversalClient
	scorer     Scorer
}

func NewMicroBatcher(rdb redis.UniversalClient, scorer Scorer) *MicroBatcher {
	return &MicroBatcher{
		eventsChan: make(chan *campaignmodel.Event, 10000),
		rdb:        rdb,
		scorer:     scorer,
	}
}

func (m *MicroBatcher) Enqueue(evt *campaignmodel.Event, msgID string) {
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
	if m.scorer == nil || m.rdb == nil {
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

	for i, score := range scores {
		fraudScore := ProbabilityToFraudScore(score)
		if fraudScore >= 30 {
			key := fmt.Sprintf("ml:score:boost:%s", keys[i].CampaignID)
			err := m.rdb.Set(ctx, key, fraudScore, 30*time.Second).Err()
			if err != nil {
				slog.Error("failed to set micro-batch ml score boost to redis", "error", err, "campaign", keys[i].CampaignID)
			} else {
				metrics.MicroBatchBoostsWrittenTotal.Inc()
			}
		}
	}
}
