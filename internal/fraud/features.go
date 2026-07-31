package fraud

import "time"

const featureVectorDims = 16

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

func (featureRow *FeatureRow) ToVectorInto(buf []float64) {
	events := float64(featureRow.Events)
	clicks := float64(featureRow.Clicks)
	uniqueUsers := float64(featureRow.UniqueUsers)
	uniqueUAs := float64(featureRow.UniqueUAs)
	spendNorm := float64(featureRow.SpendMicro) / 1e6

	ctr := safeRatio(clicks, events)
	spendRatio := safeRatio(float64(featureRow.SpendMicro), float64(featureRow.BudgetLimitMicro))

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

func (featureRow *FeatureRow) ToVector() []float64 {
	buf := make([]float64, featureVectorDims)
	featureRow.ToVectorInto(buf)
	return buf
}
