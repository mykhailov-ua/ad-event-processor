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
}

type FlowPath struct {
	Weight  int32
	Landers []FlowLanderEntry
	Offers  []FlowOfferEntry
}

type FlowPathSnapshot struct {
	Paths []FlowPath
}

// FlowRouter holds an immutable flow/path snapshot swapped via RCU.
// GM-M0 ships selection only; GM-M3 wires campaign registry reload.
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

// Select chooses path, lander, and offer deterministically from userID.
// Zero allocations when userID is a fixed []byte view.
func (r *FlowRouter) Select(userID []byte) (sel FlowSelection, ok bool) {
	snap := r.active.Load()
	sel, _, ok = SelectSnapshot(snap, userID)
	return sel, ok
}

// SelectSnapshot performs weighted flow selection on an immutable snapshot.
func SelectSnapshot(snap *FlowPathSnapshot, userID []byte) (sel FlowSelection, landerURL []byte, ok bool) {
	if snap == nil || len(snap.Paths) == 0 {
		return FlowSelection{}, nil, false
	}
	pathIdx, path := selectWeightedFlow(snap.Paths, fnv1a32(userID))
	if pathIdx < 0 {
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

func selectWeightedFlow(paths []FlowPath, bucket uint32) (int, FlowPath) {
	var total int32
	for i := range paths {
		total += paths[i].Weight
	}
	if total <= 0 {
		return 0, paths[0]
	}
	target := int32(bucket % uint32(total))
	var acc int32
	for i := range paths {
		acc += paths[i].Weight
		if target < acc {
			return i, paths[i]
		}
	}
	last := len(paths) - 1
	return last, paths[last]
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
	if len(offers) == 1 {
		return 0, offers[0]
	}
	var total int32
	for i := range offers {
		total += offers[i].Weight
	}
	if total <= 0 {
		return 0, offers[0]
	}
	target := int32(bucket % uint32(total))
	var acc int32
	for i := range offers {
		acc += offers[i].Weight
		if target < acc {
			return i, offers[i]
		}
	}
	last := len(offers) - 1
	return last, offers[last]
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
