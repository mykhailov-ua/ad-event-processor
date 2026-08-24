package postback

import (
	"testing"

	"ad-event-processor/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSynthesizeEventSourceURL(t *testing.T) {
	cid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	pb := PostbackPayload{
		CampaignID: cid,
		ClickID:    "clk-99",
		GCLID:      "GCLID99",
		FBCLID:     "FB99",
	}
	pb.subSlots[0] = "px"
	got := synthesizeEventSourceURL(pb, "trk.example.com")
	require.Contains(t, got, "https://trk.example.com/click?")
	require.Contains(t, got, "campaign_id=550e8400-e29b-41d4-a716-446655440000")
	require.Contains(t, got, "click_id=clk-99")
	require.Contains(t, got, "gclid=GCLID99")
	require.Contains(t, got, "fbclid=FB99")
	require.Contains(t, got, "sub1=px")
	require.NotContains(t, got, "upstream")
}

func TestBuildPostbackPayloadFromEvent_EventSourceURL(t *testing.T) {
	cid := uuid.New()
	cust := uuid.New()
	evt := &domain.Event{
		ClickID:    "clk-proxy",
		CampaignID: cid,
		Type:       "conversion",
		Payload:    []byte(`{"gclid":"GCLID99","sub1":"px"}`),
	}
	pb := buildPostbackPayloadFromEvent(evt, cust)
	require.Equal(t, "GCLID99", pb.GCLID)
	require.Contains(t, pb.EventSourceURL, "/click?")
	require.Contains(t, pb.EventSourceURL, "click_id=clk-proxy")
	require.Contains(t, pb.EventSourceURL, "gclid=GCLID99")
}
