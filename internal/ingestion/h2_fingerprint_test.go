package ingestion

import (
	"bytes"
	"context"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestH2SettingsAnomaly_holdoutEnablePush(t *testing.T) {
	flags := h2WireFlagSettings | h2WireFlagEnablePush
	assert.True(t, h2SettingsAnomaly(chromeDesktopUA, flags, 1, h2ChromeInitialWindow, 0))
	assert.False(t, h2SettingsAnomaly(chromeDesktopUA, flags, 0, h2ChromeInitialWindow, 0))
}

func TestH2SettingsAnomaly_holdoutGoInitialWindow(t *testing.T) {
	flags := h2WireFlagSettings
	assert.True(t, h2SettingsAnomaly(chromeDesktopUA, flags, 0, 65535, 0))
	assert.False(t, h2SettingsAnomaly(chromeDesktopUA, flags, 0, h2ChromeInitialWindow, 0))
}

func TestH2PseudoOrder_holdoutChromeVsFirefox(t *testing.T) {
	assert.False(t, h2PseudoOrderMismatch(chromeDesktopUA, h2PseudoOrderChrome, 4))
	assert.True(t, h2PseudoOrderMismatch(chromeDesktopUA, h2PseudoOrderFirefox, 4))
	assert.False(t, h2PseudoOrderMismatch("curl/8.0", h2PseudoOrderFirefox, 4))
}

func TestH2DowngradeArtifact_holdoutConnectionHeader(t *testing.T) {
	var req parsedHTTPRequest
	var hFlags uint8
	cl := 0
	require.NoError(t, h2AssignHeader(&req, []byte("connection"), []byte("keep-alive"), &hFlags, &cl))
	assert.True(t, h2DowngradeArtifact(req.H2WireFlags))
}

func TestH2Ingress_captureChromeFingerprint(t *testing.T) {
	settings := encodeH2SettingsFrame([][2]uint32{
		{uint32(h2SettingEnablePush), 0},
		{uint32(h2SettingInitialWindowSize), h2ChromeInitialWindow},
	})
	winUpd := encodeH2WindowUpdateFrame(0, h2ChromeWindowUpdate)
	body := []byte(`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}`)
	wire := buildH2TrackWire(body, settings, winUpd)

	st := newH2ConnState()
	consumed, req, _, _, err := parseH2Ingress(wire, &st, 1<<20)
	require.NoError(t, err)
	assert.Greater(t, consumed, h2ClientPrefaceLen)
	assert.NotZero(t, req.H2SettingsCRC)
	assert.Equal(t, h2ChromeInitialWindow, req.H2InitialWindow)
	assert.Equal(t, h2ChromeWindowUpdate, req.H2WindowUpdateInc)
}

func TestL7WireFilter_h2Signals(t *testing.T) {
	f := NewL7WireFilter()

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	evt.UA = chromeDesktopUA
	evt.IngressH2 = 1
	evt.H2WireFlags = h2WireFlagSettings | h2WireFlagEnablePush
	evt.H2EnablePush = 1

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonH2SettingsMismatch))

	acc.reset()
	evt.H2WireFlags = h2WireFlagPseudo
	evt.H2PseudoOrder = h2PseudoOrderFirefox
	evt.H2PseudoOrderCount = 4
	evt.H2EnablePush = 0

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonH2PseudoOrder))

	acc.reset()
	evt.H2WireFlags = h2WireFlagDowngrade
	evt.H2PseudoOrder = h2PseudoOrderChrome

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonH2DowngradeArtifact))
}

func buildH2TrackWire(body []byte, prefaceFrames ...[]byte) []byte {
	hdrBlock := []byte{0x83, 0x04, 0x06, '/', 't', 'r', 'a', 'c', 'k'}
	if len(body) > 0 {
		hdrBlock = append(hdrBlock, 0x9f)
	}

	var headersFrame []byte
	hdr := make([]byte, 9)
	encodeH2FrameHeader(hdr, uint32(len(hdrBlock)), h2FrameHeaders, h2FlagEndHeaders, 1)
	headersFrame = append(headersFrame, hdr...)
	headersFrame = append(headersFrame, hdrBlock...)

	var dataFrame []byte
	if len(body) > 0 {
		dh := make([]byte, 9)
		encodeH2FrameHeader(dh, uint32(len(body)), h2FrameData, h2FlagEndStream, 1)
		dataFrame = append(dataFrame, dh...)
		dataFrame = append(dataFrame, body...)
	}

	var buf bytes.Buffer
	buf.WriteString("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")
	for _, chunk := range prefaceFrames {
		buf.Write(chunk)
	}
	buf.Write(headersFrame)
	buf.Write(dataFrame)
	return buf.Bytes()
}
