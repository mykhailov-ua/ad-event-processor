package ingestion

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFlowRouter_LanderWeightSplit(t *testing.T) {
	t.Parallel()
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	landerB := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	offerA := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	snap := &FlowPathSnapshot{
		Paths: []FlowPath{{
			Weight: 100,
			Landers: []FlowLanderEntry{
				{LanderID: landerA, Weight: 70, URL: []byte("https://lander-a.test/lp")},
				{LanderID: landerB, Weight: 30, URL: []byte("https://lander-b.test/lp")},
			},
			Offers: []FlowOfferEntry{{OfferID: offerA, Weight: 100}},
		}},
	}
	counts := map[uuid.UUID]int{}
	for i := 0; i < 10000; i++ {
		sel, url, ok := SelectSnapshot(snap, []byte(fmt.Sprintf("user-%d", i)))
		require.True(t, ok)
		require.NotEmpty(t, url)
		counts[sel.LanderID]++
	}
	ratio := float64(counts[landerA]) / 10000
	require.InDelta(t, 0.70, ratio, 0.05)
}

func TestClickRedirect_FlowRouter(t *testing.T) {
	cid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	landerB := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	offerA := uuid.MustParse("00000000-0000-4000-8000-000000000101")

	table := NewCampaignFlowTable()
	table.Publish(&campaignFlowRegistrySnapshot{
		byCampaign: map[uuid.UUID]FlowPathSnapshot{
			cid: {
				Paths: []FlowPath{{
					Weight: 100,
					Landers: []FlowLanderEntry{
						{LanderID: landerA, Weight: 70, URL: []byte("https://lander-a.test/lp?cid={click_id}")},
						{LanderID: landerB, Weight: 30, URL: []byte("https://lander-b.test/lp?cid={click_id}")},
					},
					Offers: []FlowOfferEntry{{OfferID: offerA, Weight: 100}},
				}},
			},
		},
	})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	h.ConfigureCampaignFlow(table)

	path := "/click?campaign_id=" + cid.String() + "&type=click&user_id=flow-user-42&click_id=sticky-click-1"
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
	}, nil))
	require.Equal(t, http.StatusFound, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.True(t,
		strings.Contains(resp, "https://lander-a.test/lp") || strings.Contains(resp, "https://lander-b.test/lp"),
		"redirect location: %s", resp,
	)

	_, conn2 := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
	}, nil))
	require.Equal(t, ParseGnetHTTPStatus(conn.Written()), ParseGnetHTTPStatus(conn2.Written()))
	loc1 := flowRedirectLocation(conn.Written())
	loc2 := flowRedirectLocation(conn2.Written())
	require.Equal(t, loc1, loc2)
}

func flowRedirectLocation(wire []byte) string {
	s := string(wire)
	const prefix = "Location: "
	i := strings.Index(s, prefix)
	if i < 0 {
		return ""
	}
	s = s[i+len(prefix):]
	if j := strings.Index(s, "\r\n"); j >= 0 {
		return s[:j]
	}
	return s
}
