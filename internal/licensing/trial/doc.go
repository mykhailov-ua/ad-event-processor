// Package trial implements pilot SKU trial registry checks shared with cmd/trial-registry and license-issue.
//
// Role:
//   - trial.go reads VENDOR_TRIAL_REGISTRY JSON for telegram/hwid/wallet repeat-trial defense at JWT issue time.
//   - Complements cmd/vendor-trial-bot automation; not consulted on every ingest request.
//
// Topology:
//   - Vendor-plane only paths; appliance may omit registry file.
//
// Invariants:
//   - Reject second pilot for same telegram or hwid with status active|expired unless VENDOR_TRIAL_FORCE.
//   - USDT wallet one-buyer-line rule enforced at issue, not runtime ingest.
//
// Forbidden:
//   - Trial registry network call from tracker hot path.
//
// Verify:
//
//	go test ./internal/licensing/trial/... -short -count=1
package trial
