package ingestion

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

func TestOpenRTB26ColdNotOnAuctionStack(t *testing.T) {
	// Schain alone is ~1 KiB; cold must live in connContext, not auction stack frames.
	const schainMin = 900
	if sz := int(unsafe.Sizeof(OpenRTB26Cold{}.Schain)); sz < schainMin {
		t.Fatalf("expected SchainNodes >= %d bytes, got %d", schainMin, sz)
	}
}
