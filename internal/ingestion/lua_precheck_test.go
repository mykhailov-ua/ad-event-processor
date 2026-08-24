package ingestion

import (
	"context"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestUnifiedFilter_LuaScriptSlimmed_noDeterministicGates_holdout(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		script string
	}{
		{name: "unified", script: unifiedFilterLua},
		{name: "budget_fast", script: budgetFastLua},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lower := strings.ToLower(tc.script)
			require.NotContains(t, lower, "sismember", "fraud blacklist must run in Go precheck")
			require.NotContains(t, lower, "max_rpd", "ingress RPD must run in Go precheck")
		})
	}
}

func TestUnifiedFilter_LuaScriptSlimmed_ttcGateInGoFlag(t *testing.T) {
	require.Contains(t, unifiedFilterLua, "ttc_in_go")
	require.Contains(t, unifiedFilterLua, "ARGV[34]")
}

type ingressRPDPipeTracker struct {
	mockPipeliner
	mock *ingressRPDRedisMock
}

func (p *ingressRPDPipeTracker) Incr(ctx context.Context, key string) *redis.IntCmd {
	p.mock.incrCalls++
	return p.mockPipeliner.Incr(ctx, key)
}

func (p *ingressRPDPipeTracker) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx)
	cmd.SetVal(true)
	return cmd
}

type ingressRPDRedisMock struct {
	mockRedisClient
	incrCalls int
}

func (m *ingressRPDRedisMock) Pipeline() redis.Pipeliner {
	return &ingressRPDPipeTracker{
		mock: m,
		mockPipeliner: mockPipeliner{
			incrCmd: redis.NewIntCmd(context.Background()),
			doCmd:   redis.NewCmd(context.Background()),
		},
	}
}

func TestUnifiedFilter_applyLuaGoPrechecks_ingressRPDHandledExternally_holdout(t *testing.T) {
	t.Parallel()
	ctx := attachFilterDeadline(t.Context(), time.Second)
	campID := uuid.New()
	custID := uuid.New()
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	campInfo := &domain.Campaign{
		ID:         campID,
		CustomerID: custID,
	}
	evt := &domain.Event{
		Type:       "impression",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
	}
	reg := &entitlementsTestRegistry{maxRPD: 100}
	f := &UnifiedFilter{registry: reg}
	redisClient := &ingressRPDRedisMock{}

	f.SetIngressRPDHandledExternally(true)
	require.NoError(t, f.applyLuaGoPrechecks(ctx, evt, campInfo, redisClient, now))
	require.Equal(t, 0, redisClient.incrCalls, "external ingress RPD must skip UnifiedFilter INCR")

	f.SetIngressRPDHandledExternally(false)
	require.NoError(t, f.applyLuaGoPrechecks(ctx, evt, campInfo, redisClient, now))
	require.Equal(t, 1, redisClient.incrCalls, "without external flag UnifiedFilter must INCR")
}
