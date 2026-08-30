// Package landerhost stores and serves hosted lander ZIP contents from a filesystem root.
//
// Role:
//   - Store validates zip paths, extracts under lander UUID directory, serves index.html and static assets.
//   - editor.go helpers for path-safe joins used by internal/flow hosted lander routes.
//
// Defaults and limits:
//   - DefaultMaxZipBytes 32 MiB per upload.
//
// Topology:
//   - Called from internal/flow hosted editor; stdlib os/path/filepath only.
//
// Invariants:
//   - Zip slip blocked via clean path checks under lander root.
//   - Missing index.html fails publish validation at caller.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/landerhost/... -short -count=1
package landerhost
