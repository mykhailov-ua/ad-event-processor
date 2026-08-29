package unified

import (
	"sync"

	filt "ad-event-processor/internal/filter"
)

var (
	dirtyCampaignsKeyVal = filt.StringVal{S: "budget:dirty_campaigns"}
	dirtyCustomersKeyVal = filt.StringVal{S: "budget:dirty_customers"}
	refillNeededKeyVal   = filt.StringVal{S: "budget:refill_needed"}
	fcapIgnoredKeyVal    = filt.StringVal{S: "fcap:ignored"}
)

var (
	zeroAny      = filt.ZeroAny
	oneAny       = filt.OneAny
	hourAnyCache = filt.HourAnyCache
)

type UnifiedStringWrappers struct {
	clickID     filt.StringVal
	evtType     filt.StringVal
	payload     filt.StringVal
	ip          filt.StringVal
	ua          filt.StringVal
	userID      filt.StringVal
	placementID filt.StringVal
}

type UnifiedCheckScratch struct {
	wDup, wIdem, wDate, wDS, wFcap, wImpTS, wQuota, wRefillLock, wFence, wFrozen filt.BufWrapper
	wDeadlineMono, wNowMono                                                      filt.BufWrapper
	deadlineMonoStr, nowMonoStr                                                  filt.StringVal
	args                                                                         []any
	wrappers                                                                     UnifiedStringWrappers
	keyVals                                                                      [unifiedFilterKeyCount]filt.StringVal
	keyArgs                                                                      [unifiedFilterKeyCount]any
}

var UnifiedScratchPool = sync.Pool{
	New: func() any {
		s := &UnifiedCheckScratch{
			args: make([]any, 35),
		}
		s.wDup.Buf = make([]byte, 0, 128)
		s.wIdem.Buf = make([]byte, 0, 128)
		s.wDate.Buf = make([]byte, 0, 128)
		s.wDS.Buf = make([]byte, 0, 128)
		s.wFcap.Buf = make([]byte, 0, 128)
		s.wImpTS.Buf = make([]byte, 0, 128)
		s.wQuota.Buf = make([]byte, 0, 128)
		s.wRefillLock.Buf = make([]byte, 0, 128)
		s.wFence.Buf = make([]byte, 0, 128)
		s.wFrozen.Buf = make([]byte, 0, 128)
		s.wDeadlineMono.Buf = make([]byte, 0, 24)
		s.wNowMono.Buf = make([]byte, 0, 24)
		for i := range s.keyVals {
			s.keyArgs[i] = &s.keyVals[i]
		}
		return s
	},
}

func (s *UnifiedCheckScratch) Acquire() {}
func (s *UnifiedCheckScratch) Release() {}
