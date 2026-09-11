package ingest

import (
	"net/http"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBudgetFailover_redirectsNotDebits_holdout(t *testing.T) {
	cid := uuid.New()
	brandID := uuid.New()
	WithStaticCampaign(func(campPtr **domain.Campaign) {
		*campPtr = &domain.Campaign{
			ID:                 cid,
			CustomerID:         uuid.Nil,
			BrandID:            &brandID,
			BudgetFailoverMode: string(domain.BudgetFailoverModeFallbackURL),
			FallbackClickURL:   "https://fallback.test/offer",
		}
	})
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil, 0)
	store.SetFixturesForTest(map[uuid.UUID][]BrandCreativeFixture{brandID: {{
		URL:    "https://lander.test/go?cid={click_id}",
		Weight: 100,
	}}})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	engine := NewFilterEngine(0, &errFilter{err: ErrBudgetExhausted})
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, engine, nil, nil, NewJumpHashSharder(1), "fraud-stream", store)

	path := "/click?campaign_id=" + cid.String() + "&type=click&click_id=failover-hold&user_id=u1"
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil))
	require.Equal(t, http.StatusFound, ParseGnetHTTPStatus(conn.Written()))
	require.Contains(t, string(conn.Written()), "fallback.test/offer")
}

func TestBudgetFailover_noneReturns402_holdout(t *testing.T) {
	cid := uuid.New()
	brandID := uuid.New()
	WithStaticCampaign(func(campPtr **domain.Campaign) {
		*campPtr = &domain.Campaign{
			ID:         cid,
			CustomerID: uuid.Nil,
			BrandID:    &brandID,
		}
	})
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil, 0)
	store.SetFixturesForTest(map[uuid.UUID][]BrandCreativeFixture{brandID: {{
		URL:    "https://lander.test/go?cid={click_id}",
		Weight: 100,
	}}})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	engine := NewFilterEngine(0, &errFilter{err: ErrBudgetExhausted})
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, engine, nil, nil, NewJumpHashSharder(1), "fraud-stream", store)

	path := "/click?campaign_id=" + cid.String() + "&type=click&click_id=budget-reject&user_id=u1"
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil))
	require.Equal(t, http.StatusPaymentRequired, ParseGnetHTTPStatus(conn.Written()))
}
