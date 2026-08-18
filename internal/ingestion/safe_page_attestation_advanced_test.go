package ingestion

import (
	"encoding/json"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testCanvasHash64 = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	testAudioHash64  = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
)

func validAdvancedFingerprint() safePageVerifyFingerprint {
	return safePageVerifyFingerprint{
		UA:                     "Mozilla/5.0",
		Lang:                   "en",
		Languages:              []string{"en"},
		Timezone:               "America/New_York",
		WebRTCLocalIP:          "192.168.1.10",
		Mobile:                 true,
		CanvasHash:             testCanvasHash64,
		AudioHash:              testAudioHash64,
		NotificationPermission: "denied",
		NotificationQuery:      "denied",
	}
}

func humanMouseEvents(n int) []safePageVerifyEvent {
	out := make([]safePageVerifyEvent, n)
	for i := range out {
		out[i] = safePageVerifyEvent{
			T:  "mousemove",
			TS: int64(i * 12),
			X:  i*7 + (i % 3),
			Y:  i*5 + (i % 2),
		}
	}
	return out
}

func linearMouseEvents(n int) []safePageVerifyEvent {
	out := make([]safePageVerifyEvent, n)
	for i := range out {
		out[i] = safePageVerifyEvent{
			T:  "mousemove",
			TS: int64(i * 10),
			X:  i * 10,
			Y:  i * 10,
		}
	}
	return out
}

func TestSafePageAttestationAdvanced_permissionsMismatch(t *testing.T) {
	fp := validAdvancedFingerprint()
	fp.NotificationPermission = "granted"
	fp.NotificationQuery = "denied"
	fail, code := evaluateSafePageAttestation(safePageAttestationInput{
		fingerprint: fp,
		events:      humanMouseEvents(10),
	})
	require.True(t, fail)
	require.Equal(t, safePageAttestPermissionsMismatch, code)
}

func TestSafePageAttestationAdvanced_linearBezierReject(t *testing.T) {
	fail, code := evaluateSafePageAttestation(safePageAttestationInput{
		fingerprint: validAdvancedFingerprint(),
		events:      linearMouseEvents(12),
	})
	require.True(t, fail)
	require.Equal(t, safePageAttestBezierBot, code)
}

func TestSafePageAttestationAdvanced_humanMousePassesBezier(t *testing.T) {
	fail, code := evaluateSafePageAttestation(safePageAttestationInput{
		fingerprint: validAdvancedFingerprint(),
		events:      humanMouseEvents(18),
	})
	require.False(t, fail)
	require.Equal(t, "", code)
}

func TestSafePageAttestationAdvanced_webglHeldOutStillFails(t *testing.T) {
	fp := validAdvancedFingerprint()
	fp.WebGLRenderer = "Google SwiftShader"
	fail, code := evaluateSafePageAttestation(safePageAttestationInput{
		fingerprint: fp,
		events:      humanMouseEvents(10),
	})
	require.True(t, fail)
	require.Equal(t, safePageAttestWebGLAutomation, code)
}

func TestSafePageVerify_linearBezier_SafeView(t *testing.T) {
	h, cid := testSafePageVerifyHandler(t)
	h.trackProc.ingestGeo = &staticGeoProvider{country: "US"}

	body, err := json.Marshal(safePageVerifyRequest{
		CampaignID:  cid.String(),
		Events:      linearMouseEvents(18),
		Fingerprint: validAdvancedFingerprint(),
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
	require.Contains(t, resp, safePageAttestBezierBot)
	require.Contains(t, resp, `safe.example/white`)
}

func TestParseSafePageVerifyRequest_maxBody(t *testing.T) {
	body := make([]byte, safePageVerifyMaxBody+1)
	_, ok := parseSafePageVerifyRequest(body)
	require.False(t, ok)
}
