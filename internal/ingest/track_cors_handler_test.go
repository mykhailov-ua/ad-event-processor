package ingest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTrackOPTIONS_GnetPreflight(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		MaxRequestBodySize: 1 << 20,
		TrackCORSOrigins:   []string{"https://lp.example"},
	}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	inbound := BuildGnetHTTP("OPTIONS", "/track", map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"Origin":         "https://lp.example",
	}, nil)
	_, conn := ServeGnetHarness(h, inbound)
	require.Equal(t, http.StatusNoContent, ParseGnetHTTPStatus(conn.Written()))
	body := string(conn.Written())
	require.Contains(t, body, "Access-Control-Allow-Origin: https://lp.example")
}

func TestTrackOPTIONS_NetHTTPPreflight(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		MaxRequestBodySize: 1 << 20,
		TrackCORSOrigins:   []string{"https://lp.example"},
	}
	router := NewRouter(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil, nil, nil)
	req := httptest.NewRequest(http.MethodOptions, "/track", http.NoBody)
	req.Header.Set("Origin", "https://lp.example")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "https://lp.example", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestTrackPOST_GnetAcceptedWithCORSOrigin(t *testing.T) {
	t.Parallel()
	origin := "http://localhost:5173"
	cfg := &config.Config{
		MaxRequestBodySize: 1 << 20,
		TrackCORSOrigins:   []string{origin},
	}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	body := []byte(`{"campaign_id":"` + uuid.NewString() + `","type":"conversion","click_id":"cors-smoke","event_id":"evt-cors-1"}`)
	inbound := BuildGnetHTTP("POST", "/track", map[string]string{
		"Content-Type": "application/json",
		"Connection":   "keep-alive",
		"Origin":       origin,
	}, body)
	_, conn := ServeGnetHarness(h, inbound)
	resp := conn.Written()
	require.Equal(t, http.StatusAccepted, ParseGnetHTTPStatus(resp))
	require.Contains(t, string(resp), "Access-Control-Allow-Origin: "+origin)
	require.Contains(t, string(resp), `"status":"accepted"`)
}

func TestGnetTrackAcceptedHeaderBudget_CORSExceedsLegacy200(t *testing.T) {
	t.Parallel()
	origin := "http://localhost:5173"
	cors := newTrackCORS([]string{origin})
	bodyLen := 74
	budget := gnetTrackAcceptedHeaderBudget(origin, cors, bodyLen, false)
	require.Greater(t, budget, 200)
	require.Greater(t, budget+bodyLen, 200+bodyLen)
}
