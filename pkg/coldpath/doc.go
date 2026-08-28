// Package coldpath: shared cold HTTP helpers (ReadLimitedBody, DecodeRequestOrBadRequest,
// UUID parse). DefaultMaxBody = 64 KiB. No internal/* imports.
//
// Verify:
//   go test ./pkg/coldpath/ -short -count=1
//   bash scripts/ci/cold_path_json_gate.sh
//
package coldpath
