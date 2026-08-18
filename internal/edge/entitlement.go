package edge

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const (
	entitlementDeploymentKey = "entitlement:deployment"
	entitlementEbpfXDPEdge   = "ebpf_xdp_edge"
)

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
