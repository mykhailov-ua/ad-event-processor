package controlplane

func calcCTR(clicks, impressions int64) float64 {
	if impressions <= 0 {
		return 0
	}
	return float64(clicks) / float64(impressions)
}

func CalcROIPct(profitMicro, spendMicro int64) float64 {
	return calcROIPct(profitMicro, spendMicro)
}

func calcROIPct(profitMicro, spendMicro int64) float64 {
	if spendMicro <= 0 {
		return 0
	}
	return float64(profitMicro) / float64(spendMicro) * 100
}

func calcIVTRate(ivtEvents, clicks int64) float64 {
	if clicks <= 0 {
		return 0
	}
	rate := float64(ivtEvents) / float64(clicks)
	if rate < 0 {
		return 0
	}
	if rate > 1 {
		return 1
	}
	return rate
}

func calcCPAMicro(spendMicro, conversions int64) int64 {
	if conversions <= 0 {
		return 0
	}
	return spendMicro / conversions
}

func CalcQualityFromDrift(pacingDriftPct float64) float64 {
	return calcQualityFromDrift(pacingDriftPct)
}

func calcQualityFromDrift(pacingDriftPct float64) float64 {
	if pacingDriftPct <= 0 {
		return 1
	}
	q := 1 - pacingDriftPct/100
	if q < 0 {
		return 0
	}
	return q
}
