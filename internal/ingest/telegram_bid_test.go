package ingest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTgBidRequest(t *testing.T) {
	t.Parallel()
	body := []byte(`{"ip":"1.2.3.4","user_agent":"Mozilla/5.0","publisher_id":"pub123","widget_id":"widget456","bid_floor":0.005,"premium":true,"motivated":false,"width":300,"height":250,"production":true}`)
	var parsed telegramBidRequest
	ok := parseTelegramBidRequest(body, &parsed)
	require.True(t, ok)
	require.Equal(t, "1.2.3.4", string(parsed.IP))
	require.Equal(t, "Mozilla/5.0", string(parsed.UA))
	require.Equal(t, "pub123", string(parsed.PublisherID))
	require.Equal(t, "widget456", string(parsed.WidgetID))
	require.InDelta(t, 0.005, parsed.BidFloor, 0.000001)
	require.True(t, parsed.Premium)
	require.False(t, parsed.Motivated)
	require.Equal(t, int32(300), parsed.Width)
	require.Equal(t, int32(250), parsed.Height)
	require.True(t, parsed.Production)
}

func BenchmarkParseTgBidRequest_ZeroAlloc(b *testing.B) {
	body := []byte(`{"ip":"1.2.3.4","user_agent":"Mozilla/5.0","publisher_id":"pub123","widget_id":"widget456","bid_floor":0.005,"premium":true,"motivated":false,"width":300,"height":250,"production":true}`)
	var parsed telegramBidRequest

	b.ReportAllocs()
	for b.Loop() {
		_ = parseTelegramBidRequest(body, &parsed)
	}
}

func FuzzDecodeTgBid(f *testing.F) {
	f.Add([]byte(`{"ip":"1.2.3.4","user_agent":"ua","publisher_id":"pub"}`))
	f.Fuzz(func(t *testing.T, body []byte) {
		if len(body) > 1024 {
			return
		}
		var parsed telegramBidRequest
		_ = parseTelegramBidRequest(body, &parsed)
	})
}
