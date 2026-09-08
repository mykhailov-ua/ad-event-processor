package ingest

import (
	"context"
	"net/http"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type debitProbeFilter struct {
	checkCalls int
}

func (f *debitProbeFilter) Check(ctx context.Context, evt *domain.Event) error {
	f.checkCalls++
	return nil
}

func (f *debitProbeFilter) StreamDeferredToProducer() bool { return true }

func (f *debitProbeFilter) SetDeferStreamToProducer(deferWrite bool) {}

func (f *debitProbeFilter) ClickAmountMicro() int64 { return 100 }

func (f *debitProbeFilter) ImpressionAmountMicro() int64 { return 1 }

func (f *debitProbeFilter) LocalQuantaFullSkipEligible(evt *domain.Event, camp *domain.Campaign) bool {
	return false
}

func (f *debitProbeFilter) RollbackDebit(ctx context.Context, evt *domain.Event, camp *domain.Campaign, debitAmount int64, isLocalQuanta bool) {
}

func (f *debitProbeFilter) FinalizeLocalQuantaPublish(ctx context.Context, evt *domain.Event, camp *domain.Campaign) error {
	return nil
}

func (f *debitProbeFilter) SetSkipBudgetDebit(skip bool) {}

func TestFilterEngine_redirectOnlySkipsUnifiedBudgetDebit_holdout(t *testing.T) {
	probe := &debitProbeFilter{}
	engine := NewFilterEngine(time.Second, probe)
	evt := &domain.Event{
		CampaignID:      uuid.New(),
		Type:            "click",
		ClickFilterTier: domain.ClickFilterTierRedirectOnly,
	}
	require.NoError(t, engine.Check(context.Background(), evt))
	require.Equal(t, 0, probe.checkCalls, "redirect_only must not invoke unified budget debit filter")
}

func TestFilterEngine_lightSkipsUnifiedBudgetDebit_holdout(t *testing.T) {
	probe := &debitProbeFilter{}
	engine := NewFilterEngine(time.Second,
		&traceFilter{name: "geo", trace: new([]string)},
		probe,
	)
	evt := &domain.Event{
		CampaignID:      uuid.New(),
		Type:            "click",
		ClickFilterTier: domain.ClickFilterTierLight,
	}
	require.NoError(t, engine.Check(context.Background(), evt))
	require.Equal(t, 0, probe.checkCalls, "light tier must not invoke unified budget debit filter")
}

func TestClickRedirectGnet_lightTierSkipsUnifiedDebit_holdout(t *testing.T) {
	probe := &debitProbeFilter{}
	cid := uuid.New()
	brandID := uuid.New()
	WithStaticCampaign(func(campPtr **domain.Campaign) {
		*campPtr = &domain.Campaign{
			ID:              cid,
			CustomerID:      uuid.Nil,
			BrandID:         &brandID,
			ClickFilterTier: "light",
		}
	})
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil, 0)
	store.SetFixturesForTest(map[uuid.UUID][]BrandCreativeFixture{brandID: {{
		URL:    "https://lander.test/go?cid={click_id}",
		Weight: 100,
	}}})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	engine := NewFilterEngine(0, probe)
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, engine, nil, nil, NewJumpHashSharder(1), "fraud-stream", store)
	path := "/click?campaign_id=" + cid.String() + "&type=click&click_id=light-tier&user_id=u1"
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil))
	require.Equal(t, http.StatusFound, ParseGnetHTTPStatus(conn.Written()))
	require.Contains(t, string(conn.Written()), "lander.test/go")
	require.Equal(t, 0, probe.checkCalls, "light tier gnet click must skip unified budget debit")
}

func TestClickRedirectGnet_lightTierLatency_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("latency holdout: run without -short for p99 gate")
	}
	cid := uuid.New()
	brandID := uuid.New()
	WithStaticCampaign(func(campPtr **domain.Campaign) {
		*campPtr = &domain.Campaign{
			ID:              cid,
			CustomerID:      uuid.Nil,
			BrandID:         &brandID,
			ClickFilterTier: "light",
		}
	})
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil, 0)
	store.SetFixturesForTest(map[uuid.UUID][]BrandCreativeFixture{brandID: {{
		URL:    "https://lander.test/go?cid={click_id}",
		Weight: 100,
	}}})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", store)
	path := "/click?campaign_id=" + cid.String() + "&type=click&click_id=lat-hold&user_id=u1"
	inbound := BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil)
	_, req, err := parseHTTP1(inbound, 1<<20, nil)
	require.NoError(t, err)
	conn := NewGnetBenchConn(inbound)
	h.React(&req, conn)

	const iterations = 200
	latencies := make([]time.Duration, iterations)
	for i := range latencies {
		conn.ClearWritten()
		conn.ClearResponses()
		start := time.Now()
		h.React(&req, conn)
		latencies[i] = time.Since(start)
		require.Equal(t, http.StatusFound, ParseGnetHTTPStatus(conn.Written()))
	}
	p99 := percentileDuration(latencies, 99)
	require.Less(t, p99, 15*time.Millisecond, "light tier in-process click redirect p99 must stay under 15 ms")
}

func TestClickRedirectGnet_fullTierLatency_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("latency holdout: run without -short for p95 gate")
	}
	h, cid, _ := setupClickRedirectHarness(t, func(c *domain.Campaign) {
		c.ClickFilterTier = "full"
	})
	engine := NewFilterEngine(0, &countingFilter{})
	h.filterEngine = engine

	path := "/click?campaign_id=" + cid.String() + "&type=click&click_id=full-lat&user_id=u1"
	inbound := BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil)
	_, req, err := parseHTTP1(inbound, 1<<20, nil)
	require.NoError(t, err)
	conn := NewGnetBenchConn(inbound)
	h.React(&req, conn)

	const iterations = 200
	latencies := make([]time.Duration, iterations)
	for i := range latencies {
		conn.ClearWritten()
		conn.ClearResponses()
		start := time.Now()
		h.React(&req, conn)
		latencies[i] = time.Since(start)
		require.Equal(t, http.StatusFound, ParseGnetHTTPStatus(conn.Written()))
	}
	p95 := percentileDuration(latencies, 95)
	require.Less(t, p95, 50*time.Millisecond, "full tier in-process click redirect p95 must stay under 50 ms without Redis debit")
}
