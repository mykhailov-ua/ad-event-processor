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

	b := unsafe.Slice((*byte)(unsafe.Pointer(cold)), unsafe.Sizeof(OpenRTB26Cold{}))
	clear(b)
}
