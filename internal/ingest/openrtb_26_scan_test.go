package ingest

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

// Holdout: without ORTB_MAX_QUOTE_CHECKS budget scan walks 1<<20 quotes; SecImp stays -1 (imp past cap).
func TestScanOpenRTB26Payload_truncatesQuoteDense(t *testing.T) {
	payload := make([]byte, 0, (1<<20)+64)
	for range 1 << 20 {
		payload = append(payload, '"')
	}
	payload = append(payload, `,"imp":[{"id":"1"}],"id":"req"}`...)

	scan := scanOpenRTB26Payload(payload)
	require.Less(t, scan.SecImp(), 0)
}

func TestScanOpenRTB26Payload_sectionsParity(t *testing.T) {
	payloads := [][]byte{
		openrtb26Sample,
		[]byte(`{"id":"r1","imp":[{"id":"1","bidfloor":1.0}],"device":{"ip":"1.1.1.1","ua":"Mozilla"},"site":{"cat":["IAB2"]},"app":{"bundle":"com.example.app"},"user":{"id":"uid-1"},"source":{"tid":"txn"},"regs":{"coppa":0}}`),
		[]byte(`{"imp":[{"id":"x"}],"dooh":{"id":"d1"}}`),
	}
	for _, payload := range payloads {
		scan := scanOpenRTB26Payload(payload)
		require.Equal(t, bytes.Index(payload, openrtbKeyImp), scan.SecImp(), "imp")
		require.Equal(t, bytes.Index(payload, openrtbKeyDevice), scan.SecDevice(), "device")
		require.Equal(t, bytes.Index(payload, openrtbKeySite), scan.SecSite(), "site")
		require.Equal(t, bytes.Index(payload, openrtbKeyApp), scan.SecApp(), "app")
		require.Equal(t, bytes.Index(payload, openrtbKeyUser), scan.SecUser(), "user")
		require.Equal(t, bytes.Index(payload, openrtbKeySource), scan.SecSource(), "source")
		require.Equal(t, bytes.Index(payload, openrtbKeyDOOH), scan.SecDOOH(), "dooh")
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
	require.Equal(t, bytes.Index(search, openrtbKeyID), scan.IdxRequestID())
	require.Equal(t, bytes.Index(search, openrtbKeyBseat), scan.IdxBseat())
	require.Equal(t, bytes.Index(payload, openrtbKeyTmax), scan.IdxTmax())
	require.Equal(t, bytes.Index(payload, openrtbKeyBidfloor), scan.IdxBidfloor())
	require.Equal(t, bytes.Index(payload, openrtbKeyDevicetype), scan.IdxDevicetype())
	require.Equal(t, bytes.Index(payload, openrtbKeyCat), scan.IdxCat())
	require.Equal(t, bytes.Index(payload, openrtbKeyWseat), scan.IdxWseat())
	require.Equal(t, bytes.Index(payload, openrtbKeySchain), scan.IdxSchain())
	require.Equal(t, bytes.Index(payload, openrtbKeyTest), scan.IdxTest())
	require.Equal(t, bytes.Index(payload, openrtbKeyMaxduration), scan.IdxMaxduration())
	require.Equal(t, bytes.Index(payload, openrtbKeyCoppa), scan.IdxCoppa())
	require.Equal(t, bytes.Index(payload, openrtbKeyGDPR), scan.IdxGDPR())
	require.Equal(t, bytes.Index(payload, openrtbKeyUSPrivacy), scan.IdxUSPrivacy())
	require.Equal(t, bytes.Index(payload, openrtbKeyCur), scan.IdxCur())
	require.Equal(t, bytes.Index(payload, openrtbKeyBidfloorcur), scan.IdxBidfloorcur())
	require.Equal(t, bytes.Index(payload, openrtbKeyBCat), scan.IdxBCat())
	require.Equal(t, bytes.Index(payload, openrtbKeyBAdv), scan.IdxBAdv())
	require.Equal(t, bytes.Index(payload, openrtbKeyBApp), scan.IdxBApp())
}

// Holdout: production scan must stay 0 allocs/op; heap escape or string concat fails this bench.
func TestScanOpenRTB26Payload_ZeroAlloc(t *testing.T) {
	payload := openrtb26Sample
	avg := testing.AllocsPerRun(100, func() {
		_ = scanOpenRTB26Payload(payload)
	})
	if avg > 0 {
		t.Fatalf("scanOpenRTB26Payload allocated %f times per run, want 0", avg)
	}
}
