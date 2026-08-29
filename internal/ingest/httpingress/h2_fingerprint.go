package httpingress

import (
	"hash/crc32"
)

const (
	h2SettingHeaderTableSize      uint16 = 0x1
	h2SettingEnablePush           uint16 = 0x2
	h2SettingMaxConcurrentStreams uint16 = 0x3
	h2SettingInitialWindowSize    uint16 = 0x4
	h2SettingMaxFrameSize         uint16 = 0x5
	h2SettingMaxHeaderListSize    uint16 = 0x6

	h2PseudoMethod    uint8 = 1
	h2PseudoAuthority uint8 = 2
	h2PseudoScheme    uint8 = 3
	h2PseudoPath      uint8 = 4

	h2PseudoOrderChrome  uint16 = 668
	h2PseudoOrderFirefox uint16 = 787

	h2ChromeInitialWindow uint32 = 6291456
	h2ChromeWindowUpdate  uint32 = 15663105

	h2WireFlagSettings   uint8 = 1 << 0
	h2WireFlagWindowUpd  uint8 = 1 << 1
	h2WireFlagPseudo     uint8 = 1 << 2
	h2WireFlagDowngrade  uint8 = 1 << 3
	h2WireFlagEnablePush uint8 = 1 << 4
)

type H2ConnFingerprint struct {
	settingsCRC      uint32
	enablePush       uint8
	enablePushSet    uint8
	initialWindow    uint32
	initialWindowSet uint8
	windowUpdateInc  uint32
	windowUpdateSet  uint8
}

func (fp *H2ConnFingerprint) CaptureSettings(payload []byte) {
	if fp == nil || len(payload) == 0 || len(payload)%6 != 0 {
		return
	}
	fp.settingsCRC = crc32.ChecksumIEEE(payload)
	for off := 0; off+6 <= len(payload); off += 6 {
		id := uint16(payload[off])<<8 | uint16(payload[off+1])
		val := uint32(payload[off+2])<<24 | uint32(payload[off+3])<<16 | uint32(payload[off+4])<<8 | uint32(payload[off+5])
		switch id {
		case h2SettingEnablePush:
			fp.enablePush = uint8(val)
			fp.enablePushSet = 1
		case h2SettingInitialWindowSize:
			fp.initialWindow = val
			fp.initialWindowSet = 1
		}
	}
}

func (fp *H2ConnFingerprint) CaptureWindowUpdate(streamID uint32, payload []byte) {
	if fp == nil || fp.windowUpdateSet != 0 || streamID != 0 || len(payload) < 4 {
		return
	}
	inc := uint32(payload[0]&0x7f)<<24 | uint32(payload[1])<<16 | uint32(payload[2])<<8 | uint32(payload[3])
	if inc == 0 {
		return
	}
	fp.windowUpdateInc = inc
	fp.windowUpdateSet = 1
}

func (fp *H2ConnFingerprint) CopyTo(req *Request) {
	if fp == nil || req == nil {
		return
	}
	if fp.settingsCRC != 0 {
		req.H2SettingsCRC = fp.settingsCRC
		req.H2WireFlags |= h2WireFlagSettings
	}
	if fp.enablePushSet != 0 {
		req.H2EnablePush = fp.enablePush
		req.H2WireFlags |= h2WireFlagEnablePush
	}
	if fp.initialWindowSet != 0 {
		req.H2InitialWindow = fp.initialWindow
	}
	if fp.windowUpdateSet != 0 {
		req.H2WindowUpdateInc = fp.windowUpdateInc
		req.H2WireFlags |= h2WireFlagWindowUpd
	}
}

func recordH2PseudoHeader(req *Request, id uint8) {
	if req == nil || req.H2PseudoOrderCount >= 4 {
		return
	}
	req.H2PseudoOrder = req.H2PseudoOrder<<3 | uint16(id&7)
	req.H2PseudoOrderCount++
	req.H2WireFlags |= h2WireFlagPseudo
}

func h2KeyHasUppercase(key []byte) bool {
	for _, c := range key {
		if c >= 'A' && c <= 'Z' {
			return true
		}
	}
	return false
}

func h2ForbiddenH1HeaderName(key []byte) bool {
	return http1KeyMatchFold(key, "connection") ||
		http1KeyMatchFold(key, "keep-alive") ||
		http1KeyMatchFold(key, "transfer-encoding") ||
		http1KeyMatchFold(key, "upgrade") ||
		http1KeyMatchFold(key, "proxy-connection")
}

func markH2DowngradeArtifact(req *Request) {
	if req != nil {
		req.H2WireFlags |= h2WireFlagDowngrade
	}
}

func H2SettingsAnomaly(ua string, flags uint8, enablePush uint8, initialWindow, windowInc uint32) bool {
	if flags&h2WireFlagEnablePush != 0 && enablePush != 0 {
		return true
	}
	if flags&h2WireFlagSettings == 0 || headerOrderPolicy.ChromeNotChromium == nil || !headerOrderPolicy.ChromeNotChromium(ua) {
		return false
	}
	if initialWindow != 0 && initialWindow != h2ChromeInitialWindow {
		switch initialWindow {
		case 4194304, 10485760:
		default:
			if initialWindow <= 65535 || initialWindow == 1048576 {
				return true
			}
		}
	}
	if flags&h2WireFlagWindowUpd != 0 && windowInc != 0 && windowInc != h2ChromeWindowUpdate {
		if windowInc < 1000000 {
			return true
		}
	}
	return false
}

func H2PseudoOrderMismatch(ua string, order uint16, count uint8) bool {
	if count != 4 || headerOrderPolicy.ChromeNotChromium == nil || !headerOrderPolicy.ChromeNotChromium(ua) {
		return false
	}
	return order != h2PseudoOrderChrome
}

func H2DowngradeArtifact(flags uint8) bool {
	return flags&h2WireFlagDowngrade != 0
}

func encodeH2SettingsFrame(pairs [][2]uint32) []byte {
	payload := make([]byte, 0, len(pairs)*6)
	for _, p := range pairs {
		id := p[0]
		val := p[1]
		payload = append(payload,
			byte(id>>8), byte(id),
			byte(val>>24), byte(val>>16), byte(val>>8), byte(val),
		)
	}
	hdr := make([]byte, 9)
	encodeH2FrameHeader(hdr, uint32(len(payload)), h2FrameSettings, 0, 0)
	return append(hdr, payload...)
}

func encodeH2WindowUpdateFrame(streamID uint32, increment uint32) []byte {
	payload := []byte{
		byte(increment >> 24), byte(increment >> 16), byte(increment >> 8), byte(increment),
	}
	hdr := make([]byte, 9)
	encodeH2FrameHeader(hdr, 4, h2FrameWindowUpdate, 0, streamID)
	return append(hdr, payload...)
}
