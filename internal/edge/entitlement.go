// Package edge provides edge entitlement configuration.
package edge

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const (
	entitlementDeploymentKey = "entitlement:deployment"
	entitlementEbpfXDPEdge   = "ebpf_xdp_edge"
)

// EbpfEdgeLicensed reports whether NIC-level XDP attach and bpf-sync are allowed.
// Missing Redis key or read error fails open (dev/lab without entitlement hash).
func EbpfEdgeLicensed(ctx context.Context, rdb redis.Cmdable) bool {
	if rdb == nil {
		return true
	}
	enabled, err := rdb.HGet(ctx, entitlementDeploymentKey, entitlementEbpfXDPEdge).Int()
	if err != nil {
		return true
	}
	return enabled == 1
}
