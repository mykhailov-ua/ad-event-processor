package rtb

// NoBidReason is the typed reject code for RunAuction / RunAuctionEval.
// Wire string via String(); metrics and OpenRTB no-bid paths map these values.
type NoBidReason uint8

const (
	NoBidNone NoBidReason = iota
	NoBidInvalidRequest
	NoBidEmptyShard // geo shard (GeoHash & 63) has zero campaigns
	NoBidCorruptCatalog
	NoBidNoCandidates // geo/target bucket miss or all rows filtered
	NoBidSpendFailed  // live RunAuction only: CheckAndSpendAll CAS rejected winner price
	NoBidPacingClosed
	NoBidDailyCapExceeded
	NoBidTimeout
	NoBidDealMismatch
	NoBidScanLimit // rankMaxScanCandidates (500) exceeded; load-shed large buckets
	NoBidPrebidIVT
	NoBidSchainInvalid
	NoBidBreakerOpen
	NoBidDaypartClosed
	NoBidFreqCapExceeded
)

func (n NoBidReason) OK() bool {
	return n == NoBidNone
}

func (n NoBidReason) String() string {
	switch n {
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
