// Package money parses decimal currency strings and formats micro-unit integers for billing DTOs.
//
// Role:
//   - MicroUnit 1_000_000 per currency unit; ParseDecimal and FormatMicro convert admin display fields.
//   - Rounding rules explicit for negative and fractional inputs.
//
// Topology:
//   - Used by billingadmin, ledger, and payment handlers; stdlib only.
//
// Invariants:
//   - ErrInvalidAmount on malformed decimal strings.
//   - Zero empty string parses to 0 micros without error.
//
// Forbidden:
//   - Import internal/* packages.
//   - float64 money math in hot path callers (use int64 micros).
//
// Verify:
//
//	go test ./pkg/money/... -short -count=1
package money
