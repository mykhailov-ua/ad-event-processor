package ingest

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func rotationClickHandler(
	t *testing.T,
	cid uuid.UUID,
	snap FlowPathSnapshot,
	rdb redis.UniversalClient,
) *AdsPacketHandler {
	t.Helper()
	table := NewCampaignFlowTable()
	table.Publish(NewCampaignFlowRegistrySnapshot(map[uuid.UUID]FlowPathSnapshot{
		cid: snap,
	}))

	brandID := uuid.New()
	store := clickHookBrandStore(t, brandID)
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.BrandID = &brandID
	})
	t.Cleanup(func() { cachedMockCamp.Store(nil) })

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	var shards []redis.UniversalClient
	if rdb != nil {
		shards = []redis.UniversalClient{rdb}
	}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, shards, NewJumpHashSharder(1), "fraud-stream", store)
	h.ConfigureCampaignFlow(table)
	return h
}

func serveRotationClick(h *AdsPacketHandler, cid uuid.UUID, clickID, userID string) *GnetHarnessConn {
	path := fmt.Sprintf(
		"/click?campaign_id=%s&type=click&user_id=%s&click_id=%s",
		cid.String(), userID, clickID,
	)
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil))
	return conn
}

func TestClickRedirect_unseenRotation_secondClickDifferentLander_holdout(t *testing.T) {
	cid := uuid.New()
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	landerB := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	offerA := uuid.MustParse("00000000-0000-4000-8000-000000000101")

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	snap := FlowPathSnapshot{
		Paths: []FlowPath{{
			Weight:       100,
			RotationMode: string(domain.RotationModeUnseen),
			Landers: []FlowLanderEntry{
				{LanderID: landerA, Weight: 50, URL: []byte("https://lander-a.test/lp?cid={click_id}")},
				{LanderID: landerB, Weight: 50, URL: []byte("https://lander-b.test/lp?cid={click_id}")},
			},
			Offers: []FlowOfferEntry{{OfferID: offerA, Weight: 100}},
		}},
	}
	h := rotationClickHandler(t, cid, snap, rdb)

	conn1 := serveRotationClick(h, cid, "unseen-click-1", "unseen-user-1")
	require.Equal(t, http.StatusFound, ParseGnetHTTPStatus(conn1.Written()))
	loc1 := flowRedirectLocation(conn1.Written())
	require.True(t,
		strings.Contains(loc1, "lander-a.test") || strings.Contains(loc1, "lander-b.test"),
		"redirect location: %s", loc1,
	)

	conn2 := serveRotationClick(h, cid, "unseen-click-1", "unseen-user-1")
	require.Equal(t, http.StatusFound, ParseGnetHTTPStatus(conn2.Written()))
	loc2 := flowRedirectLocation(conn2.Written())
	require.NotEqual(t, loc1, loc2, "unseen rotation must pick a different lander on repeat click")
}

func TestClickRedirect_fixOn_pinsLander_holdout(t *testing.T) {
	cid := uuid.New()
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	landerB := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	offerA := uuid.MustParse("00000000-0000-4000-8000-000000000101")

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	snap := FlowPathSnapshot{
		Paths: []FlowPath{{
			Weight:       100,
			RotationMode: string(domain.RotationModeFixOn),
			Landers: []FlowLanderEntry{
				{LanderID: landerA, Weight: 50, URL: []byte("https://lander-a.test/lp?cid={click_id}")},
				{LanderID: landerB, Weight: 50, URL: []byte("https://lander-b.test/lp?cid={click_id}")},
			},
			Offers: []FlowOfferEntry{{OfferID: offerA, Weight: 100}},
		}},
	}
	h := rotationClickHandler(t, cid, snap, rdb)

	conn1 := serveRotationClick(h, cid, "fixon-click-1", "fixon-user-1")
	require.Equal(t, http.StatusFound, ParseGnetHTTPStatus(conn1.Written()))
	loc1 := flowRedirectLocation(conn1.Written())

	conn2 := serveRotationClick(h, cid, "fixon-click-1", "fixon-user-1")
	require.Equal(t, http.StatusFound, ParseGnetHTTPStatus(conn2.Written()))
	loc2 := flowRedirectLocation(conn2.Written())
	require.Equal(t, loc1, loc2, "fix_on must pin lander for the same visitor key")
}

func TestClickRedirect_offerClickCap_secondClickStillRedirects_holdout(t *testing.T) {
	cid := uuid.New()
	landerID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	offerA := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	offerB := uuid.MustParse("00000000-0000-4000-8000-000000000102")

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	snap := FlowPathSnapshot{
		Paths: []FlowPath{{
			Weight: 100,
			Landers: []FlowLanderEntry{
				{LanderID: landerID, Weight: 100, URL: []byte("https://lander.test/lp?cid={click_id}")},
			},
			Offers: []FlowOfferEntry{
				{OfferID: offerA, Weight: 1000, CapClicksDaily: 1},
				{OfferID: offerB, Weight: 1, CapClicksDaily: 0},
			},
		}},
	}
	h := rotationClickHandler(t, cid, snap, rdb)

	conn1 := serveRotationClick(h, cid, "cap-click-1", "cap-user-1")
	require.Equal(t, http.StatusFound, ParseGnetHTTPStatus(conn1.Written()))

	date := time.Now().UTC().Format("2006-01-02")
	dailyKey := fmt.Sprintf("%s:clicks:daily:%s", offerA.String(), date)
	count, err := rdb.Get(context.Background(), dailyKey).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	conn2 := serveRotationClick(h, cid, "cap-click-2", "cap-user-1")
	require.Equal(t, http.StatusFound, ParseGnetHTTPStatus(conn2.Written()))
	countAfter, err := rdb.Get(context.Background(), dailyKey).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(1), countAfter, "second click must route to uncapped offer without re-incrementing offer A")
}
