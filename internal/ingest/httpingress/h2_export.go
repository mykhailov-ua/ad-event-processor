package httpingress

const (
	H2FrameHeaderSize  = h2FrameHeaderSize
	H2ClientPrefaceLen = h2ClientPrefaceLen

	H2FlagEndStream  = h2FlagEndStream
	H2FlagEndHeaders = h2FlagEndHeaders

	H2FrameData         = h2FrameData
	H2FrameHeaders      = h2FrameHeaders
	H2FrameSettings     = h2FrameSettings
	H2FrameContinuation = h2FrameContinuation

	H2MaxHeaderBlock = h2MaxHeaderBlock

	H2WireFlagSettings   = h2WireFlagSettings
	H2WireFlagWindowUpd  = h2WireFlagWindowUpd
	H2WireFlagPseudo     = h2WireFlagPseudo
	H2WireFlagDowngrade  = h2WireFlagDowngrade
	H2WireFlagEnablePush = h2WireFlagEnablePush

	H2ChromeInitialWindow = h2ChromeInitialWindow
	H2ChromeWindowUpdate  = h2ChromeWindowUpdate

	H2PseudoOrderChrome  = h2PseudoOrderChrome
	H2PseudoOrderFirefox = h2PseudoOrderFirefox

	H2SettingEnablePush        = h2SettingEnablePush
	H2SettingInitialWindowSize = h2SettingInitialWindowSize

	H3FrameHeaders  = h3FrameHeaders
	H3FrameData     = h3FrameData
	H3FrameSettings = h3FrameSettings
)

var H2ClientPreface = h2ClientPreface[:]

func DecodeH2FrameHeader(buf []byte) (H2Frame, int, error) {
	return decodeH2FrameHeader(buf)
}

func EncodeH2FrameHeader(dst []byte, length uint32, typ, flags byte, streamID uint32) int {
	return encodeH2FrameHeader(dst, length, typ, flags, streamID)
}

func H2WrapH1Response(dst []byte, streamID uint32, h1 []byte) (int, error) {
	return h2WrapH1Response(dst, streamID, h1)
}

func H2DecodeHeadersBlock(block []byte, req *Request) error {
	return h2DecodeHeadersBlock(block, req)
}

func H2AssignHeader(req *Request, key, val []byte, hFlags *uint8, clValue *int) error {
	return h2AssignHeader(req, key, val, hFlags, clValue)
}

func EncodeH2SettingsFrame(pairs [][2]uint32) []byte {
	return encodeH2SettingsFrame(pairs)
}

func EncodeH2WindowUpdateFrame(streamID uint32, increment uint32) []byte {
	return encodeH2WindowUpdateFrame(streamID, increment)
}

func ParseContentLengthStrict(b []byte) (int, bool) {
	return parseContentLengthStrict(b)
}

func TrimHTTPVal(b []byte) []byte {
	return trimHTTPVal(b)
}

func FoldKeyU32(key []byte, off int) uint32 {
	return foldKeyU32(key, off)
}
