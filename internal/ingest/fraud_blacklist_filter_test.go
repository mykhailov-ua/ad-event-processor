package ingest

import (
	"context"
	"sync/atomic"
	"testing"

	"ad-event-processor/internal/domain"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fraudSIsMemberMock struct {
	mockRedisClient
	hit        bool
	sisMemberN atomic.Int32
}

func (m *fraudSIsMemberMock) SIsMember(ctx context.Context, key string, member any) *redis.BoolCmd {
	m.sisMemberN.Add(1)
	staticBoolCmd.SetVal(m.hit)
	return staticBoolCmd
}

func setupFraudBlacklistBench(t testing.TB, blacklisted bool) (*FraudBlacklistFilter, *domain.Event, context.Context) {
	t.Helper()
	redisShards := []redis.UniversalClient{&fraudSIsMemberMock{hit: blacklisted}}
	f := NewFraudBlacklistFilter(redisShards)
	evt := domain.EventPool.Get().(*domain.Event)
	evt.Reset()
	evt.IP = "203.0.113.66"
	ctx := context.Background()
	for range 1000 {
		_ = f.Check(ctx, evt)
	}
	return f, evt, ctx
}

func TestFraudBlacklistFilter_cacheHit_zeroSISMEMBER(t *testing.T) {
	mock := &fraudSIsMemberMock{hit: true}
	f := NewFraudBlacklistFilter([]redis.UniversalClient{mock})
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	evt.IP = "203.0.113.66"
	ctx := context.Background()

	require.NoError(t, f.Check(ctx, evt))
	require.Equal(t, int32(1), mock.sisMemberN.Load())

	for range 100 {
		_ = f.Check(ctx, evt)
	}
	require.Equal(t, int32(1), mock.sisMemberN.Load(), "cache hit must not call SISMEMBER")
}

func TestFraudBlacklistFilter_cacheMissThenHit_holdout(t *testing.T) {
	t.Run("blacklisted", func(t *testing.T) {
		mock := &fraudSIsMemberMock{hit: true}
		f := NewFraudBlacklistFilter([]redis.UniversalClient{mock})
		evt := domain.EventPool.Get().(*domain.Event)
		defer domain.EventPool.Put(evt)
		evt.Reset()
		evt.IP = "198.51.100.42"
		acc := attachFraudAccumulator(evt)
		defer releaseFraudAccumulator(evt, acc)
		ctx := context.Background()

		require.NoError(t, f.Check(ctx, evt))
		require.Equal(t, int32(1), mock.sisMemberN.Load())
		assert.True(t, acc.has(FraudReasonL3Blocklist))

		acc.reset()
		require.NoError(t, f.Check(ctx, evt))
		require.Equal(t, int32(1), mock.sisMemberN.Load())
		assert.True(t, acc.has(FraudReasonL3Blocklist))
	})

	t.Run("not_blacklisted", func(t *testing.T) {
		mock := &fraudSIsMemberMock{hit: false}
		f := NewFraudBlacklistFilter([]redis.UniversalClient{mock})
		evt := domain.EventPool.Get().(*domain.Event)
		defer domain.EventPool.Put(evt)
		evt.Reset()
		evt.IP = "198.51.100.43"
		acc := attachFraudAccumulator(evt)
		defer releaseFraudAccumulator(evt, acc)
		ctx := context.Background()

		require.NoError(t, f.Check(ctx, evt))
		require.Equal(t, int32(1), mock.sisMemberN.Load())
		assert.False(t, acc.has(FraudReasonL3Blocklist))

		require.NoError(t, f.Check(ctx, evt))
		require.Equal(t, int32(1), mock.sisMemberN.Load())
		assert.False(t, acc.has(FraudReasonL3Blocklist))
	})
}

func TestParseBlacklistUpdatePayload(t *testing.T) {
	t.Run("ipv4", func(t *testing.T) {
		ip, reason, ok := parseBlacklistUpdatePayload("198.51.100.10:fraud")
		require.True(t, ok)
		require.Equal(t, "198.51.100.10", ip)
		require.Equal(t, "fraud", reason)
	})
	t.Run("ipv6", func(t *testing.T) {
		ip, reason, ok := parseBlacklistUpdatePayload("2001:db8::1:fraud")
		require.True(t, ok)
		require.Equal(t, "2001:db8::1", ip)
		require.Equal(t, "fraud", reason)
	})
}

func TestFraudBlacklistFilter_invalidateForcesRedisRecheck(t *testing.T) {
	mock := &fraudSIsMemberMock{hit: false}
	f := NewFraudBlacklistFilter([]redis.UniversalClient{mock})
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	evt.IP = "198.51.100.55"
	ctx := context.Background()

	require.NoError(t, f.Check(ctx, evt))
	require.Equal(t, int32(1), mock.sisMemberN.Load())

	mock.hit = true
	f.InvalidateIP(evt.IP)

	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	require.NoError(t, f.Check(ctx, evt))
	require.Equal(t, int32(2), mock.sisMemberN.Load())
	assert.True(t, acc.has(FraudReasonL3Blocklist))
}

type errSIsMemberMock struct {
	mockRedisClient
}

func (m *errSIsMemberMock) SIsMember(ctx context.Context, key string, member any) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx)
	cmd.SetErr(redis.ErrClosed)
	return cmd
}

func TestFraudBlacklistFilter_redisError_failOpen(t *testing.T) {
	f := NewFraudBlacklistFilter([]redis.UniversalClient{&errSIsMemberMock{}})
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	evt.IP = "203.0.113.1"
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonL3Blocklist))
}

func TestFraudBlacklistFilter_zeroAlloc(t *testing.T) {
	f, evt, ctx := setupFraudBlacklistBench(t, false)
	avg := testing.AllocsPerRun(100, func() {
		_ = f.Check(ctx, evt)
	})
	if avg > 0 {
		t.Fatalf("FraudBlacklistFilter.Check allocated %.1f times per run, want 0", avg)
	}
}
