package ingestion

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScanDeviceWindowParity(t *testing.T) {
	win := []byte(`{"ip":"1.1.1.1","ipv6":"::1","ua":"Mozilla","geo":{"country":"US","region":"CA"},"os":"Android","language":"en","ifa":"abc","lmt":1,"connectiontype":2}`)
	scan := scanDeviceWindow(win)
	require.Equal(t, bytes.Index(win, openrtbKeyIP), scan.idxIP)
	require.Equal(t, bytes.Index(win, openrtbKeyIPv6), scan.idxIPv6)
	require.Equal(t, bytes.Index(win, openrtbKeyUA), scan.idxUA)
	require.Equal(t, bytes.Index(win, openrtbKeyCountry), scan.idxCountry)
	require.Equal(t, bytes.Index(win, openrtbKeyOS), scan.idxOS)
	require.Equal(t, bytes.Index(win, openrtbKeyLanguage), scan.idxLanguage)
	require.Equal(t, bytes.Index(win, openrtbKeyRegion), scan.idxRegion)
	require.Equal(t, bytes.Index(win, openrtbKeyIFA), scan.idxIFA)
	require.Equal(t, bytes.Index(win, openrtbKeyLMT), scan.idxLMT)
	require.Equal(t, bytes.Index(win, openrtbKeyConnectiontype), scan.idxConnectiontype)
}

func TestScanImpObjectParity(t *testing.T) {
	obj := []byte(`{"id":"1","bidfloor":1.25,"banner":{"w":300,"h":250},"video":{"w":640,"h":480,"maxduration":30},"audio":{},"native":{},"secure":1,"pmp":{"deals":[{"id":"deal-a","bidfloor":2.0,"wseat":["s1"]}]}}`)
	scan := scanImpObject(obj)
	require.Equal(t, bytes.Index(obj, openrtbKeyID), scan.idxID)
	require.Equal(t, bytes.Index(obj, openrtbKeyBidfloor), scan.idxBidfloor)
	require.Equal(t, bytes.Index(obj, openrtbKeyBanner), scan.idxBanner)
	require.Equal(t, bytes.Index(obj, openrtbKeyVideo), scan.idxVideo)
	require.Equal(t, bytes.Index(obj, openrtbKeyAudio), scan.idxAudio)
	require.Equal(t, bytes.Index(obj, openrtbKeyNative), scan.idxNative)
	require.Equal(t, bytes.Index(obj, openrtbKeySecure), scan.idxSecure)
	searchFrom := bytes.Index(obj, openrtbKeyDeals)
	if searchFrom < 0 {
		searchFrom = bytes.Index(obj, openrtbKeyPmp)
	}
	slice := obj[searchFrom:]
	require.Equal(t, searchFrom+bytes.Index(slice, openrtbKeyID), scan.idxDealID)
	require.GreaterOrEqual(t, scan.idxBannerW, 0)
	require.GreaterOrEqual(t, scan.idxBannerH, 0)
	require.GreaterOrEqual(t, scan.idxVideoW, 0)
	require.GreaterOrEqual(t, scan.idxMaxduration, 0)
}

func TestScanImpWindowParity(t *testing.T) {
	win := []byte(`[{"id":"1","bidfloor":1.0,"banner":{"w":728,"h":90},"video":{"w":640,"h":480},"secure":1,"pmp":{"private":1,"deals":[{"id":"d1","bidfloor":2.5}]},"metric":[{"type":"viewability","value":0.5}]}]`)
	scan := scanImpWindow(win)
	require.GreaterOrEqual(t, scan.idxBanner, 0)
	require.GreaterOrEqual(t, scan.idxVideo, 0)
	require.GreaterOrEqual(t, scan.idxSecure, 0)
	require.GreaterOrEqual(t, scan.idxPrivate, 0)
	require.GreaterOrEqual(t, scan.idxMetric, 0)
}

func TestScanSchainNodeParity(t *testing.T) {
	obj := []byte(`{"asi":"exchange.com","sid":"seller-1","hp":1}`)
	scan := scanSchainNodeObject(obj)
	require.GreaterOrEqual(t, scan.idxAsi, 0)
	require.GreaterOrEqual(t, scan.idxSid, 0)

	payload := []byte(`{"schain":{"nodes":[{"asi":"exchange.com","sid":"seller-1"}]}}`)
	nodes := parseSchainNodesAt(payload, bytes.Index(payload, openrtbKeySchain))
	require.Equal(t, uint8(1), nodes.Count)
	require.Equal(t, uint8(len("exchange.com")), nodes.Nodes[0].ASILen)
	require.Equal(t, "exchange.com", string(nodes.Nodes[0].ASI[:nodes.Nodes[0].ASILen]))
	require.Equal(t, "seller-1", string(nodes.Nodes[0].SID[:nodes.Nodes[0].SIDLen]))
}

func TestScanOpenRTB26WindowZeroAlloc(t *testing.T) {
	win := sectionWindow(openrtb26Sample, bytes.Index(openrtb26Sample, openrtbKeyImp), 1536)
	avg := testing.AllocsPerRun(100, func() {
		_ = scanImpWindow(win)
		_ = scanDeviceWindow([]byte(`{"ip":"1.1.1.1","ua":"x"}`))
	})
	if avg > 0 {
		t.Fatalf("window scan allocated %f times per run, want 0", avg)
	}
}
