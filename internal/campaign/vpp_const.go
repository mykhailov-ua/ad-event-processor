package campaign

import "time"

const (
	vppLookbackDays        = 7
	vppCampaignSampleBatch = 200
	vppBatchTimeout        = 2 * time.Minute
)
