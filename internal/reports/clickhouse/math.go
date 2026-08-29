package clickhouse

func calcROIPct(profitMicro, spendMicro int64) float64 {
	if spendMicro <= 0 {
		return 0
	}
	return float64(profitMicro) / float64(spendMicro) * 100
}

func calcCPAMicro(spendMicro, conversions int64) int64 {
	if conversions <= 0 {
		return 0
	}
	return spendMicro / conversions
}
