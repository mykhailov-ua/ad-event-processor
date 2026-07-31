package domain

const luaMetricsSampleMask uint64 = 127

func histogramSampleMaskFromConfig(cfgVal int) uint64 {
	if cfgVal < 0 {
		return 0
	}
	if cfgVal == 0 {
		return luaMetricsSampleMask
	}
	return uint64(cfgVal)
}

func shouldSampleHistogram(seq uint64, mask uint64) bool {
	if mask == 0 {
		return true
	}
	return seq&mask == 0
}
