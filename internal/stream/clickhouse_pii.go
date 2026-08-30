package stream

import (
	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/piihash"
)

// clickhousePIIFields holds FixedString(16) hashes and salt version for CH batch columns.
// Populated once per event in insertTable before batch.Append; raw IP/UA/UserID never written.
type clickhousePIIFields struct {
	ipHash      [16]byte
	uaHash      [16]byte
	userIDHash  [16]byte
	subnetHash  [16]byte
	saltVersion uint8
}

// hashEventPII runs synchronously on the processor cold path inside ClickHouseStore.insertTable.
// piihash.Hasher uses HighwayHash-128 with per-field kind bytes (IP, UA, user id, subnet); stack
// buffer in pkg/piihash, no heap alloc per hash. FixedString16 conversion at append site is allocation-free.
//
// Column mapping (insertTable):
//   - impressions/clicks/conversions/default: ip_hash, ua_hash, pii_salt_version
//   - fraud_events: user_id_hash (col 2), ip_hash, ua_hash, pii_salt_version; silent_reject from fraudSilentRejectFlag(e)
//   - fraud_aggregate_spikes: subnet_hash only
//
// Invariant: nil hasher or nil event returns zero hashes and saltVersion 0 (CH gets empty FixedString).
//
// Verify:
// go test ./internal/stream/ -short -run TestClickHouseStore_StoreBatch_hashesPII -count=1
// go test ./internal/stream/ -short -run TestClickHouseStore_StoreBatch_fraudEventUserIDHash -count=1
// go test ./internal/stream/ -bench=BenchmarkClickhousePII_writePathOverhead -benchmem -count=1
func hashEventPII(h *piihash.Hasher, e *domain.Event) clickhousePIIFields {
	if h == nil || e == nil {
		return clickhousePIIFields{}
	}
	return clickhousePIIFields{
		ipHash:      h.HashIP(e.IP),
		uaHash:      h.HashUA(e.UA),
		userIDHash:  h.HashUserID(e.UserID),
		subnetHash:  h.HashSubnet(e.IP),
		saltVersion: h.Version(),
	}
}
