package rtb

const CTRPPMUnit uint32 = 1_000_000

func normalizeCTRPPM(ctrPPM uint32) uint32 {
	if ctrPPM == 0 {
		return CTRPPMUnit
	}
	return ctrPPM
}

func effectiveScore(bid int64, ctrPPM uint32) int64 {
	return effectiveScoreWithBoost(bid, ctrPPM, CTRPPMUnit)
}

func effectiveScoreWithBoost(bid int64, ctrPPM, boostPPM uint32) int64 {
	ctr := normalizeCTRPPM(ctrPPM)
	boost := normalizeCTRPPM(boostPPM)
	denom := int64(CTRPPMUnit) * int64(CTRPPMUnit)
	return bid * int64(ctr) * int64(boost) / denom
}
