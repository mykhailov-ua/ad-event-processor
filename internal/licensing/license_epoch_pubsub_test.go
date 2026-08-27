package licensing_test

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/licensing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestLicenseEpochPubsub_invalidatesSubscriber(t *testing.T) {
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)
	licensing.ResetGuardForTest()
	t.Cleanup(licensing.ResetGuardForTest)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		licensing.WaitLicenseEpochSyncForTest()
	})

	licensing.StartLicenseEpochSync(ctx, rdb)
	licensing.PublishFeatureSeed(0x1234_5678, true)
	require.False(t, licensing.LicenseEpochInvalid())

	licensing.InvalidateLicenseEpoch()
	require.Eventually(t, func() bool {
		return licensing.LicenseEpochInvalid() && !licensing.FeatureSeedValid()
	}, time.Second, 10*time.Millisecond)
}

func TestLicenseEpochPubsub_applyNotice(t *testing.T) {
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)
	licensing.ResetGuardForTest()
	t.Cleanup(licensing.ResetGuardForTest)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		licensing.WaitLicenseEpochSyncForTest()
	})

	licensing.StartLicenseEpochSync(ctx, rdb)
	licensing.PublishFeatureSeed(0xabcd, true)

	payload := `{"seq":42,"reason":"remote"}`
	require.NoError(t, rdb.Publish(ctx, licensing.LicenseEpochPubSubChannel, payload).Err())

	require.Eventually(t, func() bool {
		return licensing.LicenseEpochInvalid() && !licensing.FeatureSeedValid()
	}, time.Second, 10*time.Millisecond)
}
