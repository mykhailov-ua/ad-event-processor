package ingestion

import "espx/internal/rtb"

func noBidToRejectKind(reason rtb.NoBidReason) filterRejectKind {
	switch reason {
	case rtb.NoBidPacingClosed:
		return filterRejectPacing
	case rtb.NoBidDaypartClosed:
		return filterRejectSchedule
	case rtb.NoBidFreqCapExceeded:
		return filterRejectFreq
	case rtb.NoBidDailyCapExceeded, rtb.NoBidSpendFailed:
		return filterRejectBudget
	case rtb.NoBidNoCandidates, rtb.NoBidEmptyShard:
		return filterRejectBidFloor
	case rtb.NoBidCorruptCatalog, rtb.NoBidInvalidRequest:
		return filterRejectInfra
	default:
		return filterRejectBidFloor
	}
}
