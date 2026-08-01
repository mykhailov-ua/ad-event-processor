package ingestion

import (
	"espx/pkg/faultproof"

	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"espx/internal/config"
	"espx/internal/domain"
	"espx/internal/rtb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newOpenRTB26FaultHandler(t *testing.T) *AdsPacketHandler {
	t.Helper()
	store := rtb.NewBudgetStore()
	catalog := NewRtbCatalog(store, BudgetAuthorityRTB)
	winnerID := uuid.New()
	geo := GeoHashFromCountry("US")
	catalog.SyncActiveCampaigns(
		[]*domain.Campaign{{ID: winnerID, BudgetLimit: 50_000_000}},
		map[uuid.UUID]RtbCampaignInput{
			winnerID: {BidMicro: 2_000_000, DeviceMask: 7, CategoryMask: 3, GeoHash: geo, Weight: 1},
		},
	)
	cfg := &config.Config{MaxRequestBodySize: 1 << 20, RtbExchangeMultiImpMax: 10}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	h.trackProc.rtbCatalog = catalog
	h.trackProc.rtbMode = rtbModeLive
	h.trackProc.ingestGeo = &staticGeoProvider{country: "US"}
	return h
}

func postOpenRTB26Gnet(h *AdsPacketHandler, body []byte) (int, []byte) {
	wire := BuildGnetHTTP("POST", "/openrtb/bid", map[string]string{
		"Content-Type":   "application/json",
		"Content-Length": itoa(len(body)),
	}, body)
	_, conn := ServeGnetHarness(h, wire)
	return ParseGnetHTTPStatus(conn.Written()), conn.Written()
}

func TestFault_OpenRTB26_TruncatedGnetCorpus(t *testing.T) {
	h := newOpenRTB26FaultHandler(t)
	valid := validExchangeBody()

	cases := []struct {
		name       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "missing_imp_array",
			body:       []byte(`{"id":"b1","tmax":300,"site":{"page":"x"},"device":{"ip":"1.1.1.1","ua":"x"}}`),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "truncated_id_only",
			body:       []byte(`{"id":`),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "garbage_prefix",
			body:       []byte(`{not-json-at-all`),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "empty_object",
			body:       []byte(`{}`),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "valid_single_imp",
			body:       valid,
			wantStatus: http.StatusOK,
		},
		{
			name:       "multi_imp_bid",
			body:       []byte(`{"id":"b2","tmax":300,"imp":[{"id":"1","bidfloor":0.5,"banner":{"w":1,"h":1}},{"id":"2","bidfloor":1.0,"banner":{"w":1,"h":1}}],"site":{"page":"x"},"device":{"ip":"1.1.1.1","ua":"Mozilla/5.0","devicetype":2}}`),
			wantStatus: http.StatusOK,
		},
	}

	var nobid, bid, other int
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			status, resp := postOpenRTB26Gnet(h, tc.body)
			assert.Equal(t, tc.wantStatus, status)
			switch status {
			case http.StatusOK:
				bid++
				assert.Contains(t, string(resp), "seatbid")
				assert.Contains(t, string(resp), "x-openrtb-version: 2.6")
			case http.StatusNoContent:
				nobid++
				assert.NotContains(t, string(resp), "seatbid")
			default:
				other++
			}
		})
	}

	faultproof.Log(t, "openrtb26_truncated_gnet_corpus", map[string]string{
		"cases": fmt.Sprintf("%d", len(cases)),
		"bid":   fmt.Sprintf("%d", bid),
		"nobid": fmt.Sprintf("%d", nobid),
		"other": fmt.Sprintf("%d", other),
	})
}

func TestFault_OpenRTB26_ConcurrentGnetBid(t *testing.T) {
	h := newOpenRTB26FaultHandler(t)

	valid := validExchangeBody()
	truncated := []byte(`{"id":"x","imp":[{`)

	const (
		workers    = 24
		iterations = 80
	)
	var (
		panics  atomic.Uint64
		okCount atomic.Int32
	)
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		w := w
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				body := valid
				if (w+i)%3 == 0 {
					body = truncated
				}
				func() {
					defer func() {
						if r := recover(); r != nil {
							panics.Add(1)
						}
					}()
					status, _ := postOpenRTB26Gnet(h, body)
					if status == http.StatusOK || status == http.StatusNoContent {
						okCount.Add(1)
					}
				}()
			}
		}()
	}
	wg.Wait()

	require.Zero(t, panics.Load(), "concurrent OpenRTB26 gnet ingress must not panic")
	require.Equal(t, int32(workers*iterations), okCount.Load())

	faultproof.Log(t, "openrtb26_concurrent_gnet_bid", map[string]string{
		"workers":    fmt.Sprintf("%d", workers),
		"iterations": fmt.Sprintf("%d", iterations),
		"responses":  fmt.Sprintf("%d", okCount.Load()),
		"panics":     fmt.Sprintf("%d", panics.Load()),
	})
}
