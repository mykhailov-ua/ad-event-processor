package ingestion

import (
	"espx/internal/domain"

	"github.com/google/uuid"
)

const registryWorkerCacheMax = 128

type registryWorkerCacheSlot struct {
	id   uuid.UUID
	gen  uint64
	camp *domain.Campaign
	_    [64 - 16 - 8 - 8]byte
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
		slot := &r.workerCache[worker]
		if slot.id == id && slot.gen == gen && slot.camp != nil {
			return slot.camp, true
		}
	}
	info, ok := r.campaignMapSnapshot().byID[id]
	if !ok || info.campaign == nil {
		return nil, false
	}
	if worker >= 0 && worker < registryWorkerCacheMax {
		slot := &r.workerCache[worker]
		slot.id = id
		slot.gen = gen
		slot.camp = info.campaign
	}
	return info.campaign, true
}
