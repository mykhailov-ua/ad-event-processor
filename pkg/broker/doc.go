// Package broker: mmap WAL broker wire protocol root. Subpackages: client, consumer,
// protocol, log. Daemon server lives in internal/broker (cmd/broker).
//
// Verify:
//
//	bash scripts/ci/pkg_boundary_gate.sh
//	go build -o /dev/null ./cmd/broker/
package broker
