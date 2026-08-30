// Package money parses decimal currency strings and formats micro-unit integers for billing DTOs.
//
// Role:
//   - MicroUnit 1_000_000 per currency unit; ParseDecimal and FormatDecimal for admin display fields.
//   - FormatFixed2 for two-decimal display strings; APIValueFloat for legacy JSON float export.
//   - PercentBps, PercentHundredths, PercentFromFloat, ScalePPM, MulMicro for fee and pacing math.
//   - LegacyFloatToMicro and JSONAmountToMicro bridge historical float JSON inputs (cold path only).
//
// Topology:
//   - Used by billingadmin, ledger, payment, postback, costsync providers, campaign runtime, reports.
//   - internal/stream/codec aliases MicroUnitFactor to MicroUnit for spend fields in stream payloads.
//   - stdlib strconv/fmt only; no Postgres or Redis in this package.
//
// Invariants:
//   - ErrInvalidAmount on malformed decimal strings ( lone minus, non-numeric parts).
//   - Empty or whitespace-only string parses to 0 micros without error.
//   - Fractional digits beyond six decimal places are truncated (not rounded) on parse.
//   - PercentBps, PercentHundredths, PercentFromFloat, ScalePPM, MulMicro return 0 when amount or rate <= 0.
//   - LegacyFloatToMicro rejects negative, NaN, and Inf float inputs.
//   - All arithmetic stays int64 micros; no float64 money math in exported helpers except APIValueFloat export.
//
// Tradeoffs:
//   - int64 micros vs decimal library: fixed scale matches Postgres budget columns and Redis Lua argv ints.
//   - Truncate vs banker's rounding on parse: deterministic admin input; callers format for display separately.
//   - Legacy float helpers retained: OpenAPI and provider webhooks still emit floats; new code should use decimal strings.
//   - Integer percent helpers floor via integer division; sub-micro remainder dropped (acceptable for fee estimates).
//
// Forbidden:
//   - Import internal/* packages.
//   - float64 money math on tracker hot path (use int64 micros from domain/budget).
//
// Verify:
//   go test ./pkg/money/... -short -count=1
package money
