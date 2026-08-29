package ingest

import (
	"testing"
	"unsafe"
)

func TestOpenRTB26HotSize(t *testing.T) {
	const maxHot = 256
	sz := int(unsafe.Sizeof(OpenRTB26Hot{}))
	coldSz := int(unsafe.Sizeof(OpenRTB26Cold{}))
	t.Logf("OpenRTB26Hot=%d bytes OpenRTB26Cold=%d bytes", sz, coldSz)
	if sz > maxHot {
		t.Fatalf("OpenRTB26Hot size %d exceeds %d byte hot-path budget", sz, maxHot)
	}
}

func TestOpenRTB26Parsed_splitLayout(t *testing.T) {
	var p OpenRTB26Parsed
	hot := uintptr(unsafe.Pointer(&p.OpenRTB26Hot))
	cold := uintptr(unsafe.Pointer(&p.OpenRTB26Cold))
	if cold != hot+unsafe.Sizeof(OpenRTB26Hot{}) {
		t.Fatalf("OpenRTB26Cold must immediately follow OpenRTB26Hot for merged reset")
	}
}

func TestOpenRTB26ColdNotOnAuctionStack(t *testing.T) {
	const schainMin = 900
	if sz := int(unsafe.Sizeof(OpenRTB26Cold{}.Schain)); sz < schainMin {
		t.Fatalf("expected SchainNodes >= %d bytes, got %d", schainMin, sz)
	}
}
