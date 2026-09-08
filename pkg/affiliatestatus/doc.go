// Package affiliatestatus maps inbound affiliate network statuses to internal goal names.
//
// Role:
//   - Shared by integrationschema apply/sync and processor ConversionPayoutApplier schema fallback.
//   - Avoids import cycles between stream, campaign, and postback on the processor hot path.
//
// Verify:
// go test ./pkg/affiliatestatus/ -count=1

package affiliatestatus
