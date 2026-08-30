package ingest

import (
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/ingest/httpingress"
)

const (
	http1HdrNone            = httpingress.HTTP1HdrNone
	http1HdrHost            = httpingress.HTTP1HdrHost
	http1HdrConnection      = httpingress.HTTP1HdrConnection
	http1HdrSecCHUA         = httpingress.HTTP1HdrSecCHUA
	http1HdrSecCHUAMobile   = httpingress.HTTP1HdrSecCHUAMobile
	http1HdrSecCHUAPlatform = httpingress.HTTP1HdrSecCHUAPlatform
	http1HdrUpgradeInsecure = httpingress.HTTP1HdrUpgradeInsecure
	http1HdrUserAgent       = httpingress.HTTP1HdrUserAgent
	http1HdrAccept          = httpingress.HTTP1HdrAccept
	http1HdrSecFetchSite    = httpingress.HTTP1HdrSecFetchSite
	http1HdrSecFetchMode    = httpingress.HTTP1HdrSecFetchMode
	http1HdrSecFetchDest    = httpingress.HTTP1HdrSecFetchDest
	http1HdrAcceptEncoding  = httpingress.HTTP1HdrAcceptEncoding
	http1HdrAcceptLanguage  = httpingress.HTTP1HdrAcceptLanguage
	http1HdrContentType     = httpingress.HTTP1HdrContentType

	h2WireFlagSettings         = httpingress.H2WireFlagSettings
	h2WireFlagEnablePush       = httpingress.H2WireFlagEnablePush
	h2ChromeInitialWindow      = httpingress.H2ChromeInitialWindow
	h2SettingInitialWindowSize = httpingress.H2SettingInitialWindowSize
	h2SettingEnablePush        = httpingress.H2SettingEnablePush
	h2PseudoOrderChrome        = httpingress.H2PseudoOrderChrome
	h2PseudoOrderFirefox       = httpingress.H2PseudoOrderFirefox
	h2WireFlagPseudo           = httpingress.H2WireFlagPseudo
	h2WireFlagDowngrade        = httpingress.H2WireFlagDowngrade
	h2ChromeWindowUpdate       = httpingress.H2ChromeWindowUpdate
	h2FrameHeaderSize          = httpingress.H2FrameHeaderSize
	h2FlagEndStream            = httpingress.H2FlagEndStream
	h2FlagEndHeaders           = httpingress.H2FlagEndHeaders
	h2FrameData                = httpingress.H2FrameData
	h2FrameHeaders             = httpingress.H2FrameHeaders
	h2FrameSettings            = httpingress.H2FrameSettings
	h2FrameContinuation        = httpingress.H2FrameContinuation
	h2MaxHeaderBlock           = httpingress.H2MaxHeaderBlock
	h2FrameRSTStream           = httpingress.H2FrameRSTStream

	h3FrameHeaders  = httpingress.H3FrameHeaders
	h3FrameData     = httpingress.H3FrameData
	h3FrameSettings = httpingress.H3FrameSettings
)

type h2Frame = httpingress.H2Frame

var (
	encodeH2SettingsFrame     = httpingress.EncodeH2SettingsFrame
	encodeH2WindowUpdateFrame = httpingress.EncodeH2WindowUpdateFrame
)

func newH2ConnState() h2ConnState {
	return httpingress.NewH2ConnState()
}

func decodeH2FrameHeader(buf []byte) (h2Frame, int, error) {
	return httpingress.DecodeH2FrameHeader(buf)
}

func encodeH2FrameHeader(dst []byte, length uint32, typ, flags byte, streamID uint32) int {
	return httpingress.EncodeH2FrameHeader(dst, length, typ, flags, streamID)
}

func h2DecodeHeadersBlock(block []byte, req *Request) error {
	return httpingress.H2DecodeHeadersBlock(block, req)
}

func h2WrapH1Response(dst []byte, streamID uint32, h1 []byte) (int, error) {
	return httpingress.H2WrapH1Response(dst, streamID, h1)
}

func h2AssignHeader(req *Request, key, val []byte, hFlags *uint8, clValue *int) error {
	return httpingress.H2AssignHeader(req, key, val, hFlags, clValue)
}

func h2PseudoOrderMismatch(ua string, order uint16, count uint8) bool {
	return filter.H2PseudoOrderMismatch(ua, order, count)
}

func h2DowngradeArtifact(flags uint8) bool {
	return filter.H2DowngradeArtifact(flags)
}

func quicDecodeVarint(data []byte, off int) (uint64, int, error) {
	return httpingress.QuicDecodeVarint(data, off)
}

func httpTokenValid(b []byte) bool {
	return httpingress.HTTPTokenValid(b)
}

func classifyHTTP1HeaderOrderToken(key []byte) uint8 {
	return httpingress.ClassifyHTTP1HeaderOrderToken(key)
}

func http1HeaderOrderMismatch(ua string, order []uint8, count uint8, secFetchPresent uint8) bool {
	return filter.HTTP1HeaderOrderMismatch(ua, order, count, secFetchPresent)
}

func h2SettingsAnomaly(ua string, flags uint8, enablePush uint8, initialWindow, windowInc uint32) bool {
	return filter.H2SettingsAnomaly(ua, flags, enablePush, initialWindow, windowInc)
}

func http1MatchForceSafeHeader(key []byte) bool {
	return httpingress.HTTP1MatchForceSafeHeader(key)
}

func http1ForceSafeValue(val []byte) bool {
	return httpingress.HTTP1ForceSafeValue(val)
}
