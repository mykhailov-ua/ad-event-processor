package ingestion

import (
	"hash/maphash"
	"sync"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/metrics"

	"github.com/google/uuid"
)

const localTTCCapacity = 65536

type localTTCSlot struct {
	campaignID uuid.UUID
	userHash   uint64
	impTSMs    int64
	valid      bool
}

type LocalTTCCache struct {
	seed  maphash.Seed
	mu    sync.RWMutex
	slots [localTTCCapacity]localTTCSlot
}

func NewLocalTTCCache() *LocalTTCCache {
	return &LocalTTCCache{seed: maphash.MakeSeed()}
}

func (c *LocalTTCCache) userHash(campaignID uuid.UUID, userID string) uint64 {
	var h maphash.Hash
	h.SetSeed(c.seed)
	h.Write(campaignID[:])
	h.WriteString(userID)
	return h.Sum64()
}

func (c *LocalTTCCache) Record(campaignID uuid.UUID, userID string) {
	if userID == "" {
		return
	}
	uh := c.userHash(campaignID, userID)
	idx := uh % localTTCCapacity
	nowMs := time.Now().UnixMilli()
	c.mu.Lock()
	c.slots[idx] = localTTCSlot{
		campaignID: campaignID,
		userHash:   uh,
		impTSMs:    nowMs,
		valid:      true,
	}
	c.mu.Unlock()
}

type localTTCOutcome int

const (
	localTTCOK localTTCOutcome = iota
	localTTCLow
	localTTCMissingClosed
	localTTCBypass
)

func (c *LocalTTCCache) CheckClick(campaignID uuid.UUID, userID string, minMs int64, failClosed bool) localTTCOutcome {
	if userID == "" {
		if failClosed {
			return localTTCMissingClosed
		}
		return localTTCBypass
	}
	uh := c.userHash(campaignID, userID)
	idx := uh % localTTCCapacity
	c.mu.RLock()
	slot := c.slots[idx]
	c.mu.RUnlock()
	if !slot.valid || slot.campaignID != campaignID || slot.userHash != uh {
		if failClosed {
			return localTTCMissingClosed
		}
		return localTTCBypass
	}
	if time.Now().UnixMilli()-slot.impTSMs < minMs {
		return localTTCLow
	}
	return localTTCOK
}

func ttcMinMs(ttcMinMsAny any) int64 {
	switch v := ttcMinMsAny.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	default:
		return 0
	}
}

func (f *UnifiedFilter) SetLocalTTCCache(c *LocalTTCCache) {
	f.localTTC = c
}

func (f *UnifiedFilter) applyGoTTC(evt *domain.Event) {
	if f == nil || f.localTTC == nil || evt == nil || !ttcEnabled(f.ttcMinMsAny) {
		return
	}
	minMs := ttcMinMs(f.ttcMinMsAny)
	if evt.Type == "impression" && evt.UserID != "" {
		f.localTTC.Record(evt.CampaignID, evt.UserID)
		return
	}
	if evt.Type != "click" {
		return
	}
	failClosed := f.ttcFailClosedAny == oneAny
	if !failClosed && f.registry != nil {
		if camp, ok := f.getCampaign(evt); ok && camp != nil {
			if domain.ResolveAttestationMode(camp.AttestationMode, camp.AttestationEnabled).RequiresProbe() {
				failClosed = true
			}
		}
	}
	switch f.localTTC.CheckClick(evt.CampaignID, evt.UserID, minMs, failClosed) {
	case localTTCLow:
		addFraudSignal(evt, FraudReasonLowTTC)
	case localTTCMissingClosed:
		addFraudSignal(evt, FraudReasonMissingImpTS)
	case localTTCBypass:
		metrics.TTCBypassTotal.Inc()
	}
}
