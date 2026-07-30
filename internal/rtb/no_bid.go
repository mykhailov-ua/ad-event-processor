package rtb

type NoBidReason uint8

const (
	NoBidNone NoBidReason = iota
	NoBidInvalidRequest
	NoBidEmptyShard
	NoBidCorruptCatalog
	NoBidNoCandidates
	NoBidSpendFailed
	NoBidPacingClosed
	NoBidDailyCapExceeded
	NoBidTimeout
	NoBidDealMismatch
	NoBidScanLimit
	NoBidPrebidIVT
	NoBidSchainInvalid
	NoBidBreakerOpen
	NoBidDaypartClosed
	NoBidFreqCapExceeded
)

func (reason NoBidReason) OK() bool {
	return reason == NoBidNone
}

func (reason NoBidReason) String() string {
	switch reason {
	case NoBidNone:
		return "ok"
	case NoBidInvalidRequest:
		return "invalid_request"
	case NoBidEmptyShard:
		return "empty_shard"
	case NoBidCorruptCatalog:
		return "corrupt_catalog"
	case NoBidNoCandidates:
		return "no_candidates"
	case NoBidSpendFailed:
		return "spend_failed"
	case NoBidPacingClosed:
		return "pacing_closed"
	case NoBidDailyCapExceeded:
		return "daily_cap"
	case NoBidTimeout:
		return "timeout"
	case NoBidDealMismatch:
		return "deal_mismatch"
	case NoBidScanLimit:
		return "scan_limit"
	case NoBidPrebidIVT:
		return "prebid_ivt"
	case NoBidSchainInvalid:
		return "schain_invalid"
	case NoBidBreakerOpen:
		return "breaker_open"
	case NoBidDaypartClosed:
		return "daypart_closed"
	case NoBidFreqCapExceeded:
		return "freq_cap"
	default:
		return "unknown"
	}
}
