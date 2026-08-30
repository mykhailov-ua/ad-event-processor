// Package iogate shedds or batches disk-append I/O for broker WAL and mmap paths under latency budget.
//
// Role:
//   - DiskGate tracks rolling disk latency; ErrShed when over DISK_LATENCY_BUDGET_MS.
//   - writev_linux.go batches append syscalls; disk_gate.go group-commits records on interval.
//
// Defaults and limits:
//   - DefaultDiskLatencyBudget 50ms.
//   - DefaultGroupCommitRecords 64.
//   - DefaultGroupCommitInterval 100ms.
//
// Env defaults:
//   - DISK_LATENCY_BUDGET_MS overrides default budget at gate construction.
//
// Topology:
//   - Used by internal/broker server and pkg/broker/log; golang.org/x/sys/cpu for feature detect.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/iogate/... -short -count=1
//	go test ./pkg/iogate/... -short -run TestDiskGate -count=1
package iogate
