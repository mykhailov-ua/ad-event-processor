// Package faultproof provides structured fault_test log lines for chaos and fault drill CI grep.
//
// Role:
//   - Log emits fault_proof fault=<name> key=value pairs consumed by scripts/fault and CI parsers.
//   - print.go mirrors Log for non-test binaries that run fault drills.
//
// Topology:
//   - Used by internal/* fault_test.go and cmd fault harnesses; testing.TB only in Log path.
//
// Invariants:
//   - fault token must be stable across releases for grep gates (fault-tests.mdc).
//   - No side effects beyond t.Log.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/faultproof/... -short -count=1
package faultproof
