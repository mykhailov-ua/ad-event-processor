package unified

import (
	"ad-event-processor/internal/domain"
	filt "ad-event-processor/internal/filter"
	"ad-event-processor/internal/rtb"

	redis "github.com/redis/go-redis/v9"
)

func fcapPrefixHash(camp *domain.Campaign) uint64 {
	if camp == nil {
		return 0
	}
	if camp.FcapPrefixHash != 0 {
		return camp.FcapPrefixHash
	}
	h := domain.FNV64String(camp.FcapKeyPrefix)
	camp.FcapPrefixHash = h
	return h
}

func fcapLookupKey(evt *domain.Event, camp *domain.Campaign) uint64 {
	if camp == nil || camp.FreqLimit <= 0 || evt == nil || evt.UserID == "" {
		return 0
	}
	prefix := fcapPrefixHash(camp)
	user := domain.FNV64String(evt.UserID)
	return rtb.FcapLookupKey(prefix, user)
}

func (f *UnifiedFilter) tryAcquireLocalFcap(evt *domain.Event, camp *domain.Campaign) (uint64, error) {
	if f == nil || f.localFcapLedger == nil {
		return 0, nil
	}
	lookup := fcapLookupKey(evt, camp)
	if lookup == 0 {
		return 0, nil
	}
	nowSec := uint32(filt.CachedUnixSec())
	if !f.localFcapLedger.TryAcquire(lookup, uint32(camp.FreqLimit), camp.FreqWindow, nowSec) {
		return lookup, filt.ErrFreqLimitExceeded
	}
	if evt != nil {
		evt.LocalFcapLookup = lookup
	}
	return lookup, nil
}

func (f *UnifiedFilter) rollbackLocalFcap(lookup uint64) {
	if f == nil || f.localFcapLedger == nil || lookup == 0 {
		return
	}
	f.localFcapLedger.Rollback(lookup)
}

func (f *UnifiedFilter) rollbackLocalFcapForEvent(evt *domain.Event) {
	if evt == nil || evt.LocalFcapLookup == 0 {
		return
	}
	f.rollbackLocalFcap(evt.LocalFcapLookup)
	evt.LocalFcapLookup = 0
}

func (f *UnifiedFilter) scheduleFcapBumpForEvent(client redis.UniversalClient, evt *domain.Event, camp *domain.Campaign, scratch *budgetFastScratch) {
	if f == nil || client == nil || camp == nil || evt == nil || camp.FreqLimit <= 0 || evt.UserID == "" {
		return
	}
	buf := scratch.wFcapBump.Buf[:0]
	buf = append(buf, camp.FcapKeyPrefix...)
	buf = append(buf, evt.UserID...)
	assignBufKey(&f.fcapKeyVal, buf)
	f.scheduleFcapBump(client, f.fcapKeyVal.S, camp.FreqWindow)
}
