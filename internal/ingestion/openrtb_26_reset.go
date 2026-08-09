package ingestion

import "unsafe"

func resetOpenRTB26Hot(hot *OpenRTB26Hot) {
	if hot == nil {
		return
	}
	*hot = OpenRTB26Hot{}
}

func resetOpenRTB26Cold(cold *OpenRTB26Cold) {
	if cold == nil {
		return
	}
	// Zero in place — avoid *cold = OpenRTB26Cold{} composite literal (~2.3 KB stack temp).
	b := unsafe.Slice((*byte)(unsafe.Pointer(cold)), unsafe.Sizeof(OpenRTB26Cold{}))
	clear(b)
}
