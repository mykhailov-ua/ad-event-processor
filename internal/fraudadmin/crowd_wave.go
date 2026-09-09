package fraudadmin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/filter"
	"ad-event-processor/pkg/crowdwave"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type CrowdWaveHost interface {
	ModeratorCorpusHost
	CrowdWaveRedis() redis.UniversalClient
	CrowdWaveEnabled() bool
	CrowdWavePolicy() crowdwave.WavePolicy
	CrowdWaveWindow() time.Duration
}

type CrowdWaveSummaryDTO struct {
	CampaignID       string `json:"campaign_id"`
	Active           bool   `json:"active"`
	Score            uint16 `json:"score"`
	UniqueClusters   int    `json:"unique_clusters"`
	SimhashNeighbors int    `json:"simhash_neighbors"`
	EntryCount       int    `json:"entry_count"`
}

type CrowdWave struct {
	host  CrowdWaveHost
	store *filter.CrowdWaveStore
}

func NewCrowdWave(host CrowdWaveHost, store *filter.CrowdWaveStore) *CrowdWave {
	if host == nil || store == nil {
		return nil
	}
	return &CrowdWave{host: host, store: store}
}

func (w *CrowdWave) GetSummary(ctx context.Context, campaignIDStr string) (CrowdWaveSummaryDTO, error) {
	if w == nil || w.store == nil {
		return CrowdWaveSummaryDTO{}, fmt.Errorf("crowd wave store not configured")
	}
	campaignID, err := uuid.Parse(strings.TrimSpace(campaignIDStr))
	if err != nil || campaignID == uuid.Nil {
		return CrowdWaveSummaryDTO{}, ValidationError("invalid campaign_id")
	}
	state, err := w.store.Snapshot(ctx, campaignID)
	if err != nil {
		return CrowdWaveSummaryDTO{}, err
	}
	return CrowdWaveSummaryDTO{
		CampaignID:       campaignID.String(),
		Active:           state.Active,
		Score:            state.Score,
		UniqueClusters:   state.UniqueClusters,
		SimhashNeighbors: state.SimhashNeighbors,
		EntryCount:       state.EntryCount,
	}, nil
}

func (w *CrowdWave) ExportActiveWaves(ctx context.Context) (int, error) {
	if w == nil || w.host == nil || w.store == nil || w.host.ModeratorCorpusPool() == nil {
		return 0, nil
	}
	if !w.host.CrowdWaveEnabled() {
		return 0, nil
	}
	corpus := NewModeratorCorpus(w.host)
	if corpus == nil {
		return 0, fmt.Errorf("moderator corpus not configured")
	}
	ids, err := w.store.ListCampaignIDs(ctx)
	if err != nil {
		return 0, err
	}
	states, err := w.store.SnapshotBatch(ctx, ids)
	if err != nil {
		return 0, err
	}
	upserts := make([]ModeratorCorpusUpsertRequest, 0, len(states))
	for campaignID, state := range states {
		if !state.Active {
			continue
		}
		note := fmt.Sprintf("crowd_wave:%s:score=%d:uniq=%d", campaignID.String(), state.Score, state.UniqueClusters)
		upserts = append(upserts, ModeratorCorpusUpsertRequest{
			JA3:    "crowd_wave_" + campaignID.String(),
			Note:   note,
			Source: "crowd_wave",
		})
	}
	if len(upserts) == 0 {
		return 0, nil
	}
	exported, err := corpus.UpsertTuples(ctx, upserts)
	if err != nil {
		return exported, err
	}
	return exported, nil
}
