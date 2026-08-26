package ingestion

import (
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSafePageAttestation_canvasRetestIgnoredWhenDisabled(t *testing.T) {
	fp := validAdvancedFingerprint()
	fp.CanvasHashA = testCanvasHash64
	fp.CanvasHashB = testCanvasHashAlt
	fail, code := evaluateSafePageAttestation(safePageAttestationInput{
		fingerprint:         fp,
		events:              humanMouseEvents(18),
		behaviorScore:       safePageVerifyMinEvents + 3,
		canvasRetestEnabled: false,
	})
	require.False(t, fail)
	require.Equal(t, "", code)
}

func TestSafePageAttestation_webrtcLeakDesktop(t *testing.T) {
	fail, code := evaluateSafePageAttestation(safePageAttestationInput{
		remoteIP: "8.8.8.8",
		fingerprint: safePageVerifyFingerprint{
			Mobile:                 false,
			CanvasHash:             testCanvasHash64,
			AudioHash:              testAudioHash64,
			NotificationPermission: "denied",
			NotificationQuery:      "denied",
		},
	})
	require.True(t, fail)
	require.Equal(t, safePageAttestWebRTCLeak, code)
}

func TestSafePageAttestation_webrtcPublicMatchesRemote(t *testing.T) {
	fail, code := evaluateSafePageAttestation(safePageAttestationInput{
		remoteIP: "8.8.8.8",
		fingerprint: safePageVerifyFingerprint{
			WebRTCLocalIP:          "8.8.8.8",
			Mobile:                 false,
			CanvasHash:             testCanvasHash64,
			AudioHash:              testAudioHash64,
			NotificationPermission: "denied",
			NotificationQuery:      "denied",
		},
	})
	require.True(t, fail)
	require.Equal(t, safePageAttestWebRTCLeak, code)
}

func TestSafePageAttestation_timezoneMismatch(t *testing.T) {
	fail, code := evaluateSafePageAttestation(safePageAttestationInput{
		country: "US",
		fingerprint: safePageVerifyFingerprint{
			Timezone:               "Europe/Moscow",
			WebRTCLocalIP:          "192.168.1.5",
			Mobile:                 true,
			CanvasHash:             testCanvasHash64,
			AudioHash:              testAudioHash64,
			NotificationPermission: "denied",
			NotificationQuery:      "denied",
		},
		nowUnix: time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC).Unix(),
	})
	require.True(t, fail)
	require.Equal(t, safePageAttestTimezoneSpoof, code)
}

func TestSafePageAttestation_webglAutomation(t *testing.T) {
	fail, code := evaluateSafePageAttestation(safePageAttestationInput{
		fingerprint: safePageVerifyFingerprint{
			WebGLRenderer:          "Google SwiftShader",
			WebRTCLocalIP:          "10.0.0.2",
			Mobile:                 true,
			CanvasHash:             testCanvasHash64,
			AudioHash:              testAudioHash64,
			NotificationPermission: "denied",
			NotificationQuery:      "denied",
		},
	})
	require.True(t, fail)
	require.Equal(t, safePageAttestWebGLAutomation, code)
}

func TestSafePageVerify_WebRTCLeak_SafeView(t *testing.T) {
	h, cid := testSafePageVerifyHandler(t)
	h.trackProc.ingestGeo = &staticGeoProvider{country: "US"}

	body, err := json.Marshal(safePageVerifyRequest{
		CampaignID: cid.String(),
		Events:     humanMouseEvents(18),
		Fingerprint: safePageVerifyFingerprint{
			UA:                     "Mozilla/5.0",
			Lang:                   "en",
			Languages:              []string{"en"},
			Timezone:               "America/New_York",
			Mobile:                 false,
			CanvasHash:             testCanvasHash64,
			AudioHash:              testAudioHash64,
			NotificationPermission: "denied",
			NotificationQuery:      "denied",
		},
	})
	require.NoError(t, err)

	inbound := BuildGnetHTTP("POST", safePageVerifyPath, map[string]string{
		"Content-Type": "application/json",
		"Connection":   "keep-alive",
	}, body)
	_, req, err := parseHTTP1(inbound, 1<<20, nil)
	require.NoError(t, err)
	conn := NewGnetBenchConn(inbound)
	conn.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 1234})
	h.React(req, conn)

	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, `"success":true`)
	require.Contains(t, resp, safePageAttestWebRTCLeak)
	require.Contains(t, resp, `safe.example/white`)
}

func TestSafePageVerify_TimezoneMismatch_SafeView(t *testing.T) {
	h, cid := testSafePageVerifyHandler(t)
	h.trackProc.ingestGeo = &staticGeoProvider{country: "US"}

	fp := validAdvancedFingerprint()
	fp.Timezone = "Europe/Moscow"
	fp.WebRTCLocalIP = "10.0.0.4"
	body, err := json.Marshal(safePageVerifyRequest{
		CampaignID:  cid.String(),
		Events:      humanMouseEvents(18),
		Fingerprint: fp,
	})
	require.NoError(t, err)

	inbound := BuildGnetHTTP("POST", safePageVerifyPath, map[string]string{
		"Content-Type": "application/json",
		"Connection":   "keep-alive",
	}, body)
	_, req, err := parseHTTP1(inbound, 1<<20, nil)
	require.NoError(t, err)
	conn := NewGnetBenchConn(inbound)
	h.React(req, conn)

	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, `"success":true`)
	require.Contains(t, resp, safePageAttestTimezoneSpoof)
	require.Contains(t, resp, `safe.example/white`)
}

func TestSafePageVerify_cleanMoneyPage(t *testing.T) {
	h, cid := testSafePageVerifyHandler(t)
	h.trackProc.ingestGeo = &staticGeoProvider{country: "US"}

	body, err := json.Marshal(safePageVerifyRequest{
		CampaignID:  cid.String(),
		Events:      humanMouseEvents(18),
		Fingerprint: validAdvancedFingerprint(),
	})
	require.NoError(t, err)

	inbound := BuildGnetHTTP("POST", safePageVerifyPath, map[string]string{
		"Content-Type": "application/json",
	}, body)
	_, req, err := parseHTTP1(inbound, 1<<20, nil)
	require.NoError(t, err)
	conn := NewGnetBenchConn(inbound)
	h.React(req, conn)

	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, `"success":true`)
	require.Contains(t, resp, `money.example`)
}

func testSafePageVerifyHandler(t *testing.T) (*AdsPacketHandler, uuid.UUID) {
	t.Helper()
	cid := benchClickCampaignID
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.BrandID = &benchClickBrandID
		c.SafePageEnabled = true
		c.SafePageURL = "https://safe.example/white"
	})
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil, 0)
	store.cache.Store(&brandCreativeMapSnapshot{
		byBrand: map[uuid.UUID][]brandCreativeEntry{
			benchClickBrandID: brandCreativeEntriesReady([]brandCreativeEntry{{
				URL:    "https://money.example/lp?cid={click_id}",
				Weight: 100,
			}}),
		},
	})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	engine := NewFilterEngine(0, &countingFilter{})
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, engine, nil, nil, NewJumpHashSharder(1), "fraud-stream", store)
	return h, cid
}
