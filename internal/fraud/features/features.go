package features

import "time"

const featureVectorDims = 16

const FeatureVectorDims = featureVectorDims

type FeatureRow struct {
	WindowStart      time.Time
	IPAddress        string
	CampaignID       string
	Events           uint64
	Clicks           uint64
	SpendMicro       int64
	BudgetLimitMicro int64
	UniqueUsers      uint64
	UniqueUAs        uint64
}

func safeRatio(numerator, denominator float64) float64 {
	if denominator <= 0 {
		return 0
	}
	return numerator / denominator
}

func (fr *FeatureRow) ToVectorInto(buf []float64) {
	events := float64(fr.Events)
	clicks := float64(fr.Clicks)
	uniqueUsers := float64(fr.UniqueUsers)
	uniqueUAs := float64(fr.UniqueUAs)
	spendNorm := float64(fr.SpendMicro) / 1e6

	ctr := safeRatio(clicks, events)
	spendRatio := safeRatio(float64(fr.SpendMicro), float64(fr.BudgetLimitMicro))

	buf[0] = events
	buf[1] = clicks
	buf[2] = ctr
	buf[3] = spendNorm
	buf[4] = spendRatio
	buf[5] = uniqueUsers
	buf[6] = uniqueUAs
	buf[7] = safeRatio(events, uniqueUAs)
	buf[8] = safeRatio(clicks, uniqueUAs)
	buf[9] = safeRatio(uniqueUsers, uniqueUAs)
	buf[10] = safeRatio(clicks, uniqueUsers)
	buf[11] = safeRatio(spendNorm, clicks)
	buf[12] = safeRatio(uniqueUAs, events)
	buf[13] = safeRatio(events, uniqueUsers)
	buf[14] = safeRatio(events, clicks+1)
	buf[15] = safeRatio(uniqueUsers, clicks+1)
}

func (fr *FeatureRow) ToVector() []float64 {
	buf := make([]float64, featureVectorDims)
	fr.ToVectorInto(buf)
	return buf
}
