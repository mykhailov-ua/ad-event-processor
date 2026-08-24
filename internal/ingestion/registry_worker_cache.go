package ingestion

import (
	"sync/atomic"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

const registryWorkerCacheMax = 128

type registryWorkerCacheEntry struct {
	id   uuid.UUID
	gen  uint64
	camp *domain.Campaign
}

type registryWorkerCacheSlot struct {
	ptr atomic.Pointer[registryWorkerCacheEntry]
}

func (r *Registry) storeCampaignSnapshot(s *campaignMapSnapshot) {
	if r == nil {
		return
	}
	r.data.Store(s)
	r.snapGen.Add(1)
}

func (r *Registry) GetCampaignWorker(worker int, id uuid.UUID) (*domain.Campaign, bool) {
	if r == nil {
		return nil, false
	}
	gen := r.snapGen.Load()
	if worker >= 0 && worker < registryWorkerCacheMax {
		if ent := r.workerCache[worker].ptr.Load(); ent != nil && ent.id == id && ent.gen == gen && ent.camp != nil {
			return ent.camp, true
		}
	}
	info, ok := r.campaignMapSnapshot().byID[id]
	if !ok || info.campaign == nil {
		return nil, false
	}
	if worker >= 0 && worker < registryWorkerCacheMax {
		r.workerCache[worker].ptr.Store(&registryWorkerCacheEntry{
			id:   id,
			gen:  gen,
			camp: info.campaign,
		})
	}
	return info.campaign, true
}
