package httpingress

import (
	"errors"
	"unsafe"
)

func ParseH2Ingress(buf []byte, st *H2ConnState, maxBody int64) (consumed int, req Request, streamID uint32, settingsOut []byte, err error) {
	off := 0
	n := len(buf)

	if !st.Established {
		if n < h2ClientPrefaceLen {
			return 0, req, 0, nil, ErrIncomplete
		}
		if !IsH2ClientPreface(buf) {
			return 0, req, 0, nil, ErrInvalid
		}
		off = h2ClientPrefaceLen
		st.Established = true
		if !st.SettingsSent {
			st.SettingsLen = copy(st.SettingsScratch[:], h2ConnBootstrap)
			settingsOut = st.SettingsScratch[:st.SettingsLen]
			st.SettingsSent = true
		}
	}

	for off < n {
		fr, frameLen, ferr := decodeH2FrameHeader(buf[off:])
		if ferr != nil {
			if st.SettingsSent && off == h2ClientPrefaceLen && errors.Is(ferr, ErrIncomplete) {
				return off, req, 0, settingsOut, ErrIncomplete
			}
			return off, req, 0, settingsOut, ferr
		}

		switch fr.Type {
		case h2FrameSettings:
			if fr.Flags&0x1 == 0 && len(fr.Payload) > 0 {
				st.fp.CaptureSettings(fr.Payload)
			}
			if fr.Flags&0x1 == 0 {
				settingsOut = st.appendSettingsOut(h2SettingsACK)
			}
		case h2FrameHeaders:
			if fr.StreamID == 0 {
				return off + frameLen, req, 0, settingsOut, ErrInvalid
			}
			if len(st.HeaderBlock) > 0 && fr.StreamID != st.HeaderStreamID {
				return off + frameLen, req, 0, settingsOut, ErrInvalid
			}
			st.HeaderStreamID = fr.StreamID
			if err := st.appendHeaderBlock(fr.Payload); err != nil {
				return off + frameLen, req, 0, settingsOut, err
			}
			if fr.Flags&h2FlagEndHeaders != 0 {
				if err := h2DecodeHeadersBlock(st.HeaderBlock, &req); err != nil {
					return off + frameLen, req, 0, settingsOut, err
				}
				st.fp.CopyTo(&req)
				st.HeaderBlock = st.HeaderBlock[:0]
				if fr.Flags&h2FlagEndStream != 0 {
					return off + frameLen, req, fr.StreamID, settingsOut, nil
				}
				st.ExpectData = true
				st.DataStreamID = fr.StreamID
			}
		case h2FrameContinuation:
			if fr.StreamID != st.HeaderStreamID {
				return off + frameLen, req, 0, settingsOut, ErrInvalid
			}
			if err := st.appendHeaderBlock(fr.Payload); err != nil {
				return off + frameLen, req, 0, settingsOut, err
			}
			if fr.Flags&h2FlagEndHeaders != 0 {
				if err := h2DecodeHeadersBlock(st.HeaderBlock, &req); err != nil {
					return off + frameLen, req, 0, settingsOut, err
				}
				st.fp.CopyTo(&req)
				st.HeaderBlock = st.HeaderBlock[:0]
				if fr.Flags&h2FlagEndStream != 0 {
					return off + frameLen, req, fr.StreamID, settingsOut, nil
				}
				st.ExpectData = true
				st.DataStreamID = fr.StreamID
			}
		case h2FrameData:
			if !st.ExpectData || fr.StreamID != st.DataStreamID {
				off += frameLen
				continue
			}
			if int64(len(fr.Payload)) > maxBody {
				return off + frameLen, req, 0, settingsOut, ErrPayloadTooLarge
			}
			req.Body = fr.Payload
			req.ContentLength = len(fr.Payload)
			req.HasContentLength = true
			st.ResetStream()
			return off + frameLen, req, fr.StreamID, settingsOut, nil
		case h2FramePing, h2FrameWindowUpdate:
			if fr.Type == h2FrameWindowUpdate {
				st.fp.CaptureWindowUpdate(fr.StreamID, fr.Payload)
			}
		case h2FramePriority, h2FrameRSTStream, h2FrameGoAway, h2FramePushPromise:
			return off + frameLen, req, 0, settingsOut, ErrInvalid
		default:
			return off + frameLen, req, 0, settingsOut, ErrInvalid
		}
		off += frameLen
	}
	return off, req, 0, settingsOut, ErrIncomplete
}

const (
	h2FrameHeaderSize  = 9
	h2ClientPrefaceLen = 24

	h2FlagEndStream  byte = 0x1
	h2FlagEndHeaders byte = 0x4
)

const (
	h2FrameData         byte = 0x0
	h2FrameHeaders      byte = 0x1
	h2FramePriority     byte = 0x2
	h2FrameRSTStream    byte = 0x3
	h2FrameSettings     byte = 0x4
	h2FramePushPromise  byte = 0x5
	h2FramePing         byte = 0x6
	h2FrameGoAway       byte = 0x7
	h2FrameWindowUpdate byte = 0x8
	h2FrameContinuation byte = 0x9
)

var h2ClientPreface = [24]byte{
	'P', 'R', 'I', ' ', '*', ' ', 'H', 'T', 'T', 'P', '/', '2', '.', '0', '\r', '\n',
	'\r', '\n', 'S', 'M', '\r', '\n', '\r', '\n',
}

type H2Frame struct {
	Length   uint32
	Type     byte
	Flags    byte
	StreamID uint32
	Payload  []byte
}

func IsH2ClientPreface(buf []byte) bool {
	if len(buf) < h2ClientPrefaceLen {
		return false
	}
	return *(*[24]byte)(unsafe.Pointer(&buf[0])) == h2ClientPreface
}

func decodeH2FrameHeader(buf []byte) (H2Frame, int, error) {
	var fr H2Frame
	if len(buf) < h2FrameHeaderSize {
		return fr, 0, ErrIncomplete
	}
	_ = buf[8]
	fr.Length = uint32(buf[0])<<16 | uint32(buf[1])<<8 | uint32(buf[2])
	fr.Type = buf[3]
	fr.Flags = buf[4]
	fr.StreamID = uint32(buf[5]&0x7f)<<24 | uint32(buf[6])<<16 | uint32(buf[7])<<8 | uint32(buf[8])
	total := h2FrameHeaderSize + int(fr.Length)
	if len(buf) < total {
		return fr, 0, ErrIncomplete
	}
	if fr.Length > 0 {
		fr.Payload = buf[h2FrameHeaderSize:total]
	}
	return fr, total, nil
}

func encodeH2FrameHeader(dst []byte, length uint32, typ, flags byte, streamID uint32) int {
	dst[0] = byte(length >> 16)
	dst[1] = byte(length >> 8)
	dst[2] = byte(length)
	dst[3] = typ
	dst[4] = flags
	dst[5] = byte(streamID >> 24)
	dst[6] = byte(streamID >> 16)
	dst[7] = byte(streamID >> 8)
	dst[8] = byte(streamID)
	return h2FrameHeaderSize
}
