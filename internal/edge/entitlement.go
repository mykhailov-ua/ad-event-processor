package edge

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const (
	entitlementDeploymentKey = "entitlement:deployment"
	entitlementEbpfXDPEdge   = "ebpf_xdp_edge"
)

func EbpfEdgeLicensed(ctx context.Context, redisClient redis.Cmdable) bool {
	if redisClient == nil {
		return true
	}
	enabled, err := redisClient.HGet(ctx, entitlementDeploymentKey, entitlementEbpfXDPEdge).Int()
	if err != nil {
		return true
	}
	return enabled == 1
}
