package adminapi

// calcCTR returns clicks/impressions in [0,1], or 0 when impressions <= 0.
func calcCTR(clicks, impressions int64) float64 {
	if impressions <= 0 {
		return 0
	}
	return float64(clicks) / float64(impressions)
}

// CalcROIPct returns profit/spend*100 per MetricFormulas.
func CalcROIPct(profitMicro, spendMicro int64) float64 {
	return calcROIPct(profitMicro, spendMicro)
}

// calcROIPct returns profit/spend*100 per MetricFormulas.
func calcROIPct(profitMicro, spendMicro int64) float64 {
	if spendMicro <= 0 {
		return 0
	}
	return float64(profitMicro) / float64(spendMicro) * 100
}

// calcIVTRate returns ivt_events/clicks in [0,1], or 0 when clicks <= 0.
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

// calcCPAMicro returns spend/conversions or 0.
func calcCPAMicro(spendMicro, conversions int64) int64 {
	if conversions <= 0 {
		return 0
	}
	return spendMicro / conversions
}

// CalcQualityFromDrift maps pacing drift percent to a [0,1] quality score.
func CalcQualityFromDrift(pacingDriftPct float64) float64 {
	return calcQualityFromDrift(pacingDriftPct)
}

// calcQualityFromDrift maps pacing drift percent to a [0,1] quality score.
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
