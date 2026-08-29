package cold

import (
	"strconv"

	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
)

func ExportHealthProbeMetrics(healthy bool, shardHealthy []int32) {
	if healthy {
		metrics.TrackerHealthDegraded.Set(0)
	} else {
		metrics.TrackerHealthDegraded.Set(1)
	}
	for i, st := range shardHealthy {
		v := 0.0
		if st == 1 {
			v = 1
		}
		metrics.TrackerRedisShardHealthy.WithLabelValues(strconv.Itoa(i)).Set(v)
	}
}

type CampaignTripletPick struct {
	PrimaryA int16
	PrimaryB int16
	Reserve  int16
}

func (c *CampaignTripletPick) PickShard(campaignID, userID string) int {
	if c == nil {
		return 0
	}
	id, err := uuid.Parse(campaignID)
	if err != nil {
		return int(c.PrimaryA)
	}
	hash := ComputeCompositeHashUUID(id, []byte(userID))
	pct := hash % 100
	if pct < 40 {
		return int(c.PrimaryA)
	}
	if pct < 80 {
		return int(c.PrimaryB)
	}
	return int(c.Reserve)
}
