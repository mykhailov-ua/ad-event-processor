package ingestion

import "errors"

const h2MaxHeaderBlock = 16 << 10

type h2ConnState struct {
	established            bool
	settingsSent           bool
	headerBlock            []byte
	headerStreamID         uint32
	expectData             bool
	dataStreamID           uint32
	settingsScratch        [40]byte
	settingsLen            int
	incompleteSpin         uint8
	incompleteIdleArmed    bool
	incompleteIdleDeadline int64
	fp                     h2ConnFingerprint
}

func newH2ConnState() h2ConnState {
	return h2ConnState{
		headerBlock: make([]byte, 0, 256),
	}
}

func (s *h2ConnState) resetConn() {
	s.established = false
	s.settingsSent = false
	s.settingsLen = 0
	s.incompleteSpin = 0
	s.incompleteIdleArmed = false
	s.incompleteIdleDeadline = 0
	s.fp = h2ConnFingerprint{}
	s.resetStream()
}

func (s *h2ConnState) resetStream() {
	s.headerBlock = s.headerBlock[:0]
	s.expectData = false
	s.dataStreamID = 0
	s.headerStreamID = 0
}

func (s *h2ConnState) appendSettingsOut(extra []byte) []byte {
	s.settingsLen += copy(s.settingsScratch[s.settingsLen:], extra)
	return s.settingsScratch[:s.settingsLen]
}

func (s *h2ConnState) appendHeaderBlock(p []byte) error {
	if len(s.headerBlock)+len(p) > h2MaxHeaderBlock {
		return errInvalidRequest
	}
	s.headerBlock = append(s.headerBlock, p...)
	return nil
}

func parseH2Ingress(buf []byte, st *h2ConnState, maxBody int64) (consumed int, req parsedHTTPRequest, streamID uint32, settingsOut []byte, err error) {
	off := 0
	n := len(buf)

	if !st.established {
		if n < h2ClientPrefaceLen {
			return 0, req, 0, nil, errIncompleteRequest
		}
		if !isH2ClientPreface(buf) {
			return 0, req, 0, nil, errInvalidRequest
		}
		off = h2ClientPrefaceLen
		st.established = true
		if !st.settingsSent {
			st.settingsLen = copy(st.settingsScratch[:], h2ConnBootstrap)
			settingsOut = st.settingsScratch[:st.settingsLen]
			st.settingsSent = true
		}
	}

	for off < n {
		fr, frameLen, ferr := decodeH2FrameHeader(buf[off:])
		if ferr != nil {
			if st.settingsSent && off == h2ClientPrefaceLen && errors.Is(ferr, errIncompleteRequest) {
				return off, req, 0, settingsOut, errIncompleteRequest
			}
			return off, req, 0, settingsOut, ferr
		}

		switch fr.Type {
		case h2FrameSettings:
			if fr.Flags&0x1 == 0 && len(fr.Payload) > 0 {
				st.fp.captureSettings(fr.Payload)
			}
			if fr.Flags&0x1 == 0 {
				settingsOut = st.appendSettingsOut(h2SettingsACK)
			}
		case h2FrameHeaders:
			if fr.StreamID == 0 {
				return off + frameLen, req, 0, settingsOut, errInvalidRequest
			}
			if len(st.headerBlock) > 0 && fr.StreamID != st.headerStreamID {
				return off + frameLen, req, 0, settingsOut, errInvalidRequest
			}
			st.headerStreamID = fr.StreamID
			if err := st.appendHeaderBlock(fr.Payload); err != nil {
				return off + frameLen, req, 0, settingsOut, err
			}
			if fr.Flags&h2FlagEndHeaders != 0 {
				if err := h2DecodeHeadersBlock(st.headerBlock, &req); err != nil {
					return off + frameLen, req, 0, settingsOut, err
				}
				st.fp.copyTo(&req)
				st.headerBlock = st.headerBlock[:0]
				if fr.Flags&h2FlagEndStream != 0 {
					return off + frameLen, req, fr.StreamID, settingsOut, nil
				}
				st.expectData = true
				st.dataStreamID = fr.StreamID
			}
		case h2FrameContinuation:
			if fr.StreamID != st.headerStreamID {
				return off + frameLen, req, 0, settingsOut, errInvalidRequest
			}
			if err := st.appendHeaderBlock(fr.Payload); err != nil {
				return off + frameLen, req, 0, settingsOut, err
			}
			if fr.Flags&h2FlagEndHeaders != 0 {
				if err := h2DecodeHeadersBlock(st.headerBlock, &req); err != nil {
					return off + frameLen, req, 0, settingsOut, err
				}
				st.fp.copyTo(&req)
				st.headerBlock = st.headerBlock[:0]
				if fr.Flags&h2FlagEndStream != 0 {
					return off + frameLen, req, fr.StreamID, settingsOut, nil
				}
				st.expectData = true
				st.dataStreamID = fr.StreamID
			}
		case h2FrameData:
			if !st.expectData || fr.StreamID != st.dataStreamID {
				off += frameLen
				continue
			}
			if int64(len(fr.Payload)) > maxBody {
				return off + frameLen, req, 0, settingsOut, errPayloadTooLarge
			}
			req.Body = fr.Payload
			req.ContentLength = len(fr.Payload)
			req.HasContentLength = true
			st.resetStream()
			return off + frameLen, req, fr.StreamID, settingsOut, nil
		case h2FramePing, h2FrameWindowUpdate:
			if fr.Type == h2FrameWindowUpdate {
				st.fp.captureWindowUpdate(fr.StreamID, fr.Payload)
			}
		case h2FramePriority, h2FrameRSTStream, h2FrameGoAway, h2FramePushPromise:
			return off + frameLen, req, 0, settingsOut, errInvalidRequest
		default:
			return off + frameLen, req, 0, settingsOut, errInvalidRequest
		}
		off += frameLen
	}
	return off, req, 0, settingsOut, errIncompleteRequest
}
