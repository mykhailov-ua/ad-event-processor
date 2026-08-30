// Package wal is the mmap append-only WAL for cmd/region-proxy regional ingress.
//
// Role:
//   - Open(dir, DiskWriteGate): single wal.segment file with mmap capacity default 64 MiB.
//   - Append records with seq monotonic counter; Recover discards torn tail records.
//   - Forward scanner and partition helpers for uplink/keygen/opkey consumers.
//
// Topology:
//   - Wired from internal/regionproxy server; data-dir flag on cmd/region-proxy.
//   - pkg/regionproxy/keygen and opkey read batches from WAL tail.
//
// Defaults and limits:
//   - defaultSegSize 64 MiB; ErrCorrupt, ErrSegmentFull, ErrEmptyPayload sentinels.
//   - Group commit via iogate.DiskWriteGate pressure signals.
//
// Forbidden:
//   - internal/ingest must not import pkg/regionproxy (hot path boundary).
//   - Not Postgres balance_ledger authority (global control reconciles spend).
//
// Verify:
// go test ./pkg/regionproxy/wal/... -short -count=1
// bash scripts/test/multi_region_resilience_drill.sh
package wal
