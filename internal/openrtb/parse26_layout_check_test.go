package openrtb

import (
	"testing"
	"unsafe"
)

func TestOpenRTB26Parsed_layout_holdout(t *testing.T) {
	var p OpenRTB26Parsed
	hot := uintptr(unsafe.Pointer(&p.OpenRTB26Hot))
	cold := uintptr(unsafe.Pointer(&p.OpenRTB26Cold))
	if cold != hot+unsafe.Sizeof(OpenRTB26Hot{}) {
		t.Fatalf("OpenRTB26Cold offset %d want %d", cold-hot, unsafe.Sizeof(OpenRTB26Hot{}))
	}

	type merged struct {
		OpenRTB26Hot
		OpenRTB26Cold
	}
	if unsafe.Sizeof(merged{}) != unsafe.Sizeof(OpenRTB26Parsed{}) {
		t.Fatalf("OpenRTB26Parsed size %d != merged %d", unsafe.Sizeof(OpenRTB26Parsed{}), unsafe.Sizeof(merged{}))
	}
}
