package ingestion

import (
	"sync/atomic"

	"github.com/google/uuid"
)

type FlowLanderEntry struct {
	LanderID uuid.UUID
	Weight   int32
	URL      []byte
}

type FlowOfferEntry struct {
	OfferID uuid.UUID
	Weight  int32
	URL     []byte
	Capped  bool
}

type FlowPath struct {
	Weight  int32
	Filters FlowPathFilters
	Landers []FlowLanderEntry
	Offers  []FlowOfferEntry
}

type FlowPathSnapshot struct {
	Paths []FlowPath
}

type FlowRouter struct {
	active atomic.Pointer[FlowPathSnapshot]
}

func NewFlowRouter() *FlowRouter {
	return &FlowRouter{}
}

func (r *FlowRouter) Publish(s *FlowPathSnapshot) {
	r.active.Store(s)
}

func (r *FlowRouter) Ready() bool {
	return r.active.Load() != nil
}

type FlowSelection struct {
	PathIdx   int
	LanderIdx int
	OfferIdx  int
	LanderID  uuid.UUID
	OfferID   uuid.UUID
}

func (r *FlowRouter) Select(userID []byte) (sel FlowSelection, ok bool) {
	snap := r.active.Load()
	sel, _, ok = SelectSnapshot(snap, userID, FlowSelectContext{})
	return sel, ok
}

func BanditSelect(snap *FlowPathSnapshot, userID []byte) (sel FlowSelection, landerURL []byte, ok bool) {
	return SelectSnapshot(snap, userID, FlowSelectContext{})
}

func (r *FlowRouter) BanditSelect(userID []byte) (sel FlowSelection, landerURL []byte, ok bool) {
	return BanditSelect(r.active.Load(), userID)
}

func SelectSnapshot(snap *FlowPathSnapshot, userID []byte, ctx FlowSelectContext) (sel FlowSelection, landerURL []byte, ok bool) {
	if snap == nil || len(snap.Paths) == 0 {
		return FlowSelection{}, nil, false
	}
	pathIdx, path, pathOK := selectWeightedFlowFiltered(snap.Paths, ctx, fnv1a32(userID))
	if !pathOK {
		return FlowSelection{}, nil, false
	}
	landerIdx, lander := selectWeightedLander(path.Landers, fnv1a32Salted(userID, 'l'))
	if landerIdx < 0 || len(lander.URL) == 0 {
		return FlowSelection{}, nil, false
	}
	offerIdx, offer := selectWeightedOffer(path.Offers, fnv1a32Salted(userID, 'o'))
	if offerIdx < 0 {
		return FlowSelection{}, nil, false
	}
	return FlowSelection{
		PathIdx:   pathIdx,
		LanderIdx: landerIdx,
		OfferIdx:  offerIdx,
		LanderID:  lander.LanderID,
		OfferID:   offer.OfferID,
	}, lander.URL, true
}

func selectWeightedFlowFiltered(paths []FlowPath, ctx FlowSelectContext, bucket uint32) (int, FlowPath, bool) {
	var total int32
	for i := range paths {
		if !flowPathFiltersMatch(paths[i].Filters, ctx) {
			continue
		}
		total += paths[i].Weight
	}
	if total <= 0 {
		return -1, FlowPath{}, false
	}
	target := int32(bucket % uint32(total))
	var acc int32
	for i := range paths {
		if !flowPathFiltersMatch(paths[i].Filters, ctx) {
			continue
		}
		acc += paths[i].Weight
		if target < acc {
			return i, paths[i], true
		}
	}
	for i := len(paths) - 1; i >= 0; i-- {
		if flowPathFiltersMatch(paths[i].Filters, ctx) && paths[i].Weight > 0 {
			return i, paths[i], true
		}
	}
	return -1, FlowPath{}, false
}

func selectWeightedLander(landers []FlowLanderEntry, bucket uint32) (int, FlowLanderEntry) {
	if len(landers) == 0 {
		return -1, FlowLanderEntry{}
	}
	if len(landers) == 1 {
		return 0, landers[0]
	}
	var total int32
	for i := range landers {
		total += landers[i].Weight
	}
	if total <= 0 {
		return 0, landers[0]
	}
	target := int32(bucket % uint32(total))
	var acc int32
	for i := range landers {
		acc += landers[i].Weight
		if target < acc {
			return i, landers[i]
		}
	}
	last := len(landers) - 1
	return last, landers[last]
}

func selectWeightedOffer(offers []FlowOfferEntry, bucket uint32) (int, FlowOfferEntry) {
	if len(offers) == 0 {
		return -1, FlowOfferEntry{}
	}
	var total int32
	eligible := 0
	for i := range offers {
		if offers[i].Capped || offers[i].Weight <= 0 {
			continue
		}
		total += offers[i].Weight
		eligible++
	}
	if eligible == 0 {
		return -1, FlowOfferEntry{}
	}
	if eligible == 1 {
		for i := range offers {
			if !offers[i].Capped && offers[i].Weight > 0 {
				return i, offers[i]
			}
		}
	}
	if total <= 0 {
		for i := range offers {
			if !offers[i].Capped && offers[i].Weight > 0 {
				return i, offers[i]
			}
		}
	}
	target := int32(bucket % uint32(total))
	var acc int32
	for i := range offers {
		if offers[i].Capped || offers[i].Weight <= 0 {
			continue
		}
		acc += offers[i].Weight
		if target < acc {
			return i, offers[i]
		}
	}
	for i := len(offers) - 1; i >= 0; i-- {
		if !offers[i].Capped && offers[i].Weight > 0 {
			return i, offers[i]
		}
	}
	return -1, FlowOfferEntry{}
}

func fnv1a32(b []byte) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	h := uint32(offset32)
	for _, c := range b {
		h ^= uint32(c)
		h *= prime32
	}
	return h
}

func fnv1a32Salted(userID []byte, salt byte) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	h := uint32(offset32)
	for _, c := range userID {
		h ^= uint32(c)
		h *= prime32
	}
	h ^= uint32(salt)
	h *= prime32
	return h
}
