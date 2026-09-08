package unified

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	filt "ad-event-processor/internal/filter"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

// probeRedisClient is a zero-response EVALSHA stub for alloc isolation (not production).
type probeRedisClient struct {
	redis.UniversalClient
}

func (probeRedisClient) FilterEvalFast(cmd *redis.Cmd) (int64, error) {
	return 0, nil
}

func (probeRedisClient) Process(ctx context.Context, cmd redis.Cmder) error {
	if c, ok := cmd.(*redis.Cmd); ok {
		c.SetVal(int64(0))
	}
	return nil
}

type probePipeline struct {
	redis.Pipeliner
	incrCmd *redis.IntCmd
}

func (p *probePipeline) Incr(ctx context.Context, key string) *redis.IntCmd {
	p.incrCmd.SetVal(1)
	return p.incrCmd
}

func (p *probePipeline) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx)
	cmd.SetVal(true)
	return cmd
}

func (p *probePipeline) Exec(ctx context.Context) ([]redis.Cmder, error) {
	return nil, nil
}

func (p *probeRedisClient) Pipeline() redis.Pipeliner {
	return &probePipeline{incrCmd: redis.NewIntCmd(context.Background())}
}

func probeMockFilter(t *testing.T) (*UnifiedFilter, *domain.Event, *domain.Campaign) {
	t.Helper()
	filt.ResetStaticCampaignBaseline()
	campID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	filt.ConfigureMockRegistryCampaign(func(c *domain.Campaign) {
		c.ID = campID
		c.CustomerID = uuid.Nil
		c.PacingMode = domain.PacingModeAsap
		c.Location = time.UTC
	})
	reg := &filt.MockRegistry{}
	camp, ok := reg.GetCampaign(campID)
	if !ok || camp == nil {
		t.Fatal("mock campaign setup failed")
	}
	f := NewUnifiedFilter(
		[]redis.UniversalClient{&probeRedisClient{}},
		filt.NewJumpHashSharder(1),
		reg,
		nil,
		100,
		time.Minute,
		time.Hour,
		time.Hour,
		100_000,
		10_000,
		"events",
		10_000,
	)
	f.SetLuaFastPathEnabled(true)
	evt := &domain.Event{
		Type:               "click",
		IP:                 "1.1.1.1",
		UserID:             "user123",
		CampaignID:         campID,
		ClickID:            "click123",
		FilterCamp:         camp,
		FilterCampResolved: true,
	}
	return f, evt, camp
}

func TestCheckMockAllocProbe_GetCampaign(t *testing.T) {
	reg := &filt.MockRegistry{}
	campID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	filt.ConfigureMockRegistryCampaign(func(c *domain.Campaign) {
		c.ID = campID
	})
	evt := &domain.Event{CampaignID: campID}
	for range 8 {
		_, _ = filt.GetCampaignFromEvent(reg, evt)
	}
	evt.FilterCampResolved = false
	evt.FilterCamp = nil
	allocs := testing.AllocsPerRun(100, func() {
		_, _ = filt.GetCampaignFromEvent(reg, evt)
	})
	t.Logf("GetCampaignFromEvent allocs/op: %v", allocs)
}

func TestCheckMockAllocProbe_processFilterEval(t *testing.T) {
	f, evt, _ := probeMockFilter(t)
	ctx := context.Background()
	var wireBuf [budgetFastKeyCount + budgetFastArgCount + 3]any
	wire := fillEvalShaWireN(wireBuf[:], f.fastScriptHashAny, f.budgetScratch.keyArgs[:], f.budgetScratch.args[:], budgetFastKeyCount)
	for range 8 {
		resetPooledRedisCmd(&f.evalRedisCmd, ctx, wire, 3)
		_ = f.processFilterEval(ctx, &probeRedisClient{}, 0, evt, &f.evalRedisCmd)
	}
	allocs := testing.AllocsPerRun(100, func() {
		resetPooledRedisCmd(&f.evalRedisCmd, ctx, wire, 3)
		_ = f.processFilterEval(ctx, &probeRedisClient{}, 0, evt, &f.evalRedisCmd)
	})
	t.Logf("processFilterEval allocs/op: %v (hot path uses FilterEvalFast bypass)", allocs)
}

func TestCheckMockAllocProbe_Check_fastPath(t *testing.T) {
	f, evt, _ := probeMockFilter(t)
	ctx := context.Background()
	for range 32 {
		_ = f.Check(ctx, evt)
	}
	allocs := testing.AllocsPerRun(100, func() {
		_ = f.Check(ctx, evt)
	})
	if allocs != 0 {
		t.Fatalf("Check fast-path mock want 0 allocs/op got %v", allocs)
	}
}

func TestCheckMockAllocProbe_evalShaPooledN(t *testing.T) {
	f, evt, _ := probeMockFilter(t)
	ctx := context.Background()
	keyArgs := f.budgetScratch.keyArgs[:]
	args := f.budgetScratch.args[:]
	for range 32 {
		_, _ = f.evalShaPooledN(ctx, &probeRedisClient{}, 0, evt, f.fastScriptHashAny, keyArgs, args, budgetFastKeyCount)
	}
	allocs := testing.AllocsPerRun(100, func() {
		_, _ = f.evalShaPooledN(ctx, &probeRedisClient{}, 0, evt, f.fastScriptHashAny, keyArgs, args, budgetFastKeyCount)
	})
	t.Logf("evalShaPooledN allocs/op: %v", allocs)
}

func TestCheckMockAllocProbe_runBudgetFastLua(t *testing.T) {
	f, evt, camp := probeMockFilter(t)
	ctx := context.Background()
	amount := f.clickAmountMicroAny
	scratch := &f.budgetScratch
	for range 32 {
		_ = f.runBudgetFastLua(ctx, evt, camp, amount, &probeRedisClient{}, 0, scratch)
	}
	allocs := testing.AllocsPerRun(100, func() {
		_ = f.runBudgetFastLua(ctx, evt, camp, amount, &probeRedisClient{}, 0, scratch)
	})
	t.Logf("runBudgetFastLua allocs/op: %v", allocs)
	if allocs > 1 {
		t.Fatalf("runBudgetFastLua want <=1 got %v", allocs)
	}
}

func TestCheckMockAllocProbe_fcapAsyncGoroutine(t *testing.T) {
	f, evt, camp := probeMockFilter(t)
	camp.FreqLimit = 100
	camp.FreqWindow = 3600
	ctx := context.Background()
	for range 32 {
		_ = f.Check(ctx, evt)
	}
	allocs := testing.AllocsPerRun(50, func() {
		_ = f.Check(ctx, evt)
	})
	if allocs != 0 {
		t.Fatalf("Check with FreqLimit local fcap want 0 allocs/op got %v", allocs)
	}
}
