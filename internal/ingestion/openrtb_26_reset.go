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

func resetOpenRTB26Parsed(p *OpenRTB26Parsed) {
	if p == nil {
		return
	}
	b := unsafe.Slice((*byte)(unsafe.Pointer(p)), unsafe.Sizeof(OpenRTB26Parsed{}))
	clear(b)
}

func openRTB26ParsedFromSplit(hot *OpenRTB26Hot, cold *OpenRTB26Cold) (*OpenRTB26Parsed, bool) {
	if hot == nil || cold == nil {
		return nil, false
	}
	expected := uintptr(unsafe.Pointer(hot)) + unsafe.Sizeof(OpenRTB26Hot{})
	if uintptr(unsafe.Pointer(cold)) != expected {
		return nil, false
	}
	return (*OpenRTB26Parsed)(unsafe.Pointer(hot)), true
}
