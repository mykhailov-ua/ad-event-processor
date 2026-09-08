package openrtb

import "unsafe"

type openRTB26LayoutProof struct {
	hot  OpenRTB26Hot
	cold OpenRTB26Cold
}

func openRTB26LayoutProofColdFollowsHot() {
	var p openRTB26LayoutProof
	hotEnd := uintptr(unsafe.Pointer(&p.hot)) + unsafe.Sizeof(p.hot)
	coldStart := uintptr(unsafe.Pointer(&p.cold))
	if hotEnd != coldStart {
		panic("openrtb: OpenRTB26Cold must immediately follow OpenRTB26Hot for merged reset")
	}
}

func init() {
	openRTB26LayoutProofColdFollowsHot()
}
