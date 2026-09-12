package edge

import (
	"context"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/licensing"

	"github.com/redis/go-redis/v9"
)

const (
	entitlementDeploymentKey = "entitlement:deployment"
	entitlementEbpfXDPEdge   = "ebpf_xdp_edge"
)

// EbpfEdgeSeedGateAllowed verifies license-file seed coupling for ebpf_xdp_edge when
// enterprise sealed assets are active. Dev/unsealed mode always returns true.
func EbpfEdgeSeedGateAllowed() bool {
	if config.LicenseAssetsUnsealed() {
		return true
	}
	licensing.SetSeedCouplingRequired(config.LicenseSeedCouplingEnabled())
	if !licensing.SeedCouplingRequired() {
		return true
	}
	if !config.LicenseFilePresent() {
		return false
	}
	path := config.LicensePathFromEnv()
	hostFP := licensing.HostFingerprint()
	now := time.Now().UTC()
	verified, err := licensing.VerifyLicenseFile(path, nil, hostFP, now)
	if err != nil {
		return false
	}
	if !licensing.EbpfEdgeAllowed(verified.State, verified.Entitlements) {
		return false
	}
	if licensing.LicenseEpochInvalid() {
		return false
	}
	mckWork, err := licensing.DeriveMCKWorkForRecheckFromLicenseFile(path, nil, hostFP)
	if err != nil {
		return false
	}
	seed := licensing.FeatureSeedFromMCK(mckWork)
	bits := licensing.MCKFeatureBitsFromWork(mckWork)
	licensing.PublishFeatureSeed(seed, true)
	licensing.PublishMCKFeatureBits(bits)
	return licensing.SeedGateEbpfEdge(verified.Entitlements)
}

func EbpfEdgeLicensed(ctx context.Context, redisClient redis.Cmdable) bool {
	if redisClient == nil {
		return true
	}
	enabled, err := redisClient.HGet(ctx, entitlementDeploymentKey, entitlementEbpfXDPEdge).Int()
	if err != nil {
		return false
	}
	return enabled == 1
}
