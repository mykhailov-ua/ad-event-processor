package postback

import (
	"hash/fnv"

	"github.com/google/uuid"
)

func outboundSampleBucket(clickID string, postbackID uuid.UUID) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(clickID))
	_, _ = h.Write([]byte{':'})
	_, _ = h.Write(postbackID[:])
	return h.Sum32()
}

// ShouldFireOutboundPostbackSample returns true when samplePercent is 100 or the stable
// hash bucket for (click_id, postback_id) falls below samplePercent. 0 disables firing.
func ShouldFireOutboundPostbackSample(clickID string, postbackID uuid.UUID, samplePercent int32) bool {
	if samplePercent <= 0 {
		return false
	}
	if samplePercent >= 100 {
		return true
	}
	return outboundSampleBucket(clickID, postbackID)%100 < uint32(samplePercent)
}
