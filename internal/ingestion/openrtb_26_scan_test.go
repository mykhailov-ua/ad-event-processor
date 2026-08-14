package ingestion

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScanOpenRTB26Payload_truncatesQuoteDense(t *testing.T) {
	payload := make([]byte, 0, (1<<20)+64)
	for range 1 << 20 {
		payload = append(payload, '"')
	}
	payload = append(payload, `,"imp":[{"id":"1"}],"id":"req"}`...)

	scan := scanOpenRTB26Payload(payload)
	require.Less(t, scan.sec.imp, 0)
}

func TestScanOpenRTB26Payload_sectionsParity(t *testing.T) {
	payloads := [][]byte{
		openrtb26Sample,
		[]byte(`{"id":"r1","imp":[{"id":"1","bidfloor":1.0}],"device":{"ip":"1.1.1.1","ua":"Mozilla"},"site":{"cat":["IAB2"]},"app":{"bundle":"com.example.app"},"user":{"id":"uid-1"},"source":{"tid":"txn"},"regs":{"coppa":0}}`),
		[]byte(`{"imp":[{"id":"x"}],"dooh":{"id":"d1"}}`),
	}
	for _, payload := range payloads {
		scan := scanOpenRTB26Payload(payload)
		require.Equal(t, bytes.Index(payload, openrtbKeyImp), scan.sec.imp, "imp")
		require.Equal(t, bytes.Index(payload, openrtbKeyDevice), scan.sec.device, "device")
		require.Equal(t, bytes.Index(payload, openrtbKeySite), scan.sec.site, "site")
		require.Equal(t, bytes.Index(payload, openrtbKeyApp), scan.sec.app, "app")
		require.Equal(t, bytes.Index(payload, openrtbKeyUser), scan.sec.user, "user")
		require.Equal(t, bytes.Index(payload, openrtbKeySource), scan.sec.source, "source")
		require.Equal(t, bytes.Index(payload, openrtbKeyDOOH), scan.sec.dooh, "dooh")
	}
}

func TestScanOpenRTB26Payload_topLevelParity(t *testing.T) {
	payload := []byte(`{
  "id":"req-1",
  "bseat":["blocked"],
  "tmax":250,
  "bcat":["IAB2-3"],
  "badv":["evil.com"],
  "bapp":["com.blocked"],
  "regs":{"gdpr":1,"us_privacy":"1YNN","coppa":0},
  "cur":["USD"],
  "bidfloorcur":"EUR",
  "test":1,
  "maxduration":30,
  "imp":[{"id":"1","bidfloor":1.25,"devicetype":2,"wseat":["seat-1"],"cat":["IAB1"]}],
  "source":{"ext":{"schain":{"nodes":[{"asi":"x"}]}}}
}`)
	scan := scanOpenRTB26Payload(payload)
	imp := bytes.Index(payload, openrtbKeyImp)
	search := payload
	if imp > 0 {
		search = payload[:imp]
	}
	require.Equal(t, bytes.Index(search, openrtbKeyID), scan.idxRequestID)
	require.Equal(t, bytes.Index(search, openrtbKeyBseat), scan.idxBseat)
	require.Equal(t, bytes.Index(payload, openrtbKeyTmax), scan.idxTmax)
	require.Equal(t, bytes.Index(payload, openrtbKeyBidfloor), scan.idxBidfloor)
	require.Equal(t, bytes.Index(payload, openrtbKeyDevicetype), scan.idxDevicetype)
	require.Equal(t, bytes.Index(payload, openrtbKeyCat), scan.idxCat)
	require.Equal(t, bytes.Index(payload, openrtbKeyWseat), scan.idxWseat)
	require.Equal(t, bytes.Index(payload, openrtbKeySchain), scan.idxSchain)
	require.Equal(t, bytes.Index(payload, openrtbKeyTest), scan.idxTest)
	require.Equal(t, bytes.Index(payload, openrtbKeyMaxduration), scan.idxMaxduration)
	require.Equal(t, bytes.Index(payload, openrtbKeyCoppa), scan.idxCoppa)
	require.Equal(t, bytes.Index(payload, openrtbKeyGDPR), scan.idxGDPR)
	require.Equal(t, bytes.Index(payload, openrtbKeyUSPrivacy), scan.idxUSPrivacy)
	require.Equal(t, bytes.Index(payload, openrtbKeyCur), scan.idxCur)
	require.Equal(t, bytes.Index(payload, openrtbKeyBidfloorcur), scan.idxBidfloorcur)
	require.Equal(t, bytes.Index(payload, openrtbKeyBCat), scan.idxBCat)
	require.Equal(t, bytes.Index(payload, openrtbKeyBAdv), scan.idxBAdv)
	require.Equal(t, bytes.Index(payload, openrtbKeyBApp), scan.idxBApp)
}

func TestScanOpenRTB26Payload_ZeroAlloc(t *testing.T) {
	payload := openrtb26Sample
	avg := testing.AllocsPerRun(100, func() {
		_ = scanOpenRTB26Payload(payload)
	})
	if avg > 0 {
		t.Fatalf("scanOpenRTB26Payload allocated %f times per run, want 0", avg)
	}
}
