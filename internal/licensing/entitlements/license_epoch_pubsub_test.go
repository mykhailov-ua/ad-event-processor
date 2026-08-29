package entitlements_test

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/licensing/entitlements"
	"ad-event-processor/internal/licensing/verify"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestLicenseEpochPubsub_invalidatesSubscriber(t *testing.T) {
	entitlements.ResetFeatureSeedForTest()
	t.Cleanup(entitlements.ResetFeatureSeedForTest)
	verify.ResetGuardForTest()
	t.Cleanup(verify.ResetGuardForTest)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		entitlements.WaitLicenseEpochSyncForTest()
	})

	entitlements.StartLicenseEpochSync(ctx, rdb)
	entitlements.PublishFeatureSeed(0x1234_5678, true)
	require.False(t, entitlements.LicenseEpochInvalid())

	entitlements.InvalidateLicenseEpoch()
	require.Eventually(t, func() bool {
		return entitlements.LicenseEpochInvalid() && !entitlements.FeatureSeedValid()
	}, time.Second, 10*time.Millisecond)
}

func TestLicenseEpochPubsub_applyNotice(t *testing.T) {
	entitlements.ResetFeatureSeedForTest()
	t.Cleanup(entitlements.ResetFeatureSeedForTest)
	verify.ResetGuardForTest()
	t.Cleanup(verify.ResetGuardForTest)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		entitlements.WaitLicenseEpochSyncForTest()
	})

	entitlements.StartLicenseEpochSync(ctx, rdb)
	entitlements.PublishFeatureSeed(0xabcd, true)

	payload := `{"seq":42,"reason":"remote"}`
	require.NoError(t, rdb.Publish(ctx, entitlements.LicenseEpochPubSubChannel, payload).Err())

	require.Eventually(t, func() bool {
		return entitlements.LicenseEpochInvalid() && !entitlements.FeatureSeedValid()
	}, time.Second, 10*time.Millisecond)
}
