package openrtb

type (
	OpenRTB26Scan                = openrtb26Scan
	OpenRTB26Sections            = openrtb26Sections
	OpenRTB26DeviceScan          = openrtb26DeviceScan
	OpenRTB26ImpObjScan          = openrtb26ImpObjScan
	OpenRTB26ImpWinScan          = openrtb26ImpWinScan
	OpenRTB26SupplyChainNodeScan = openrtb26SupplyChainNodeScan
)

var (
	KeyImp            = openrtbKeyImp
	KeyTmax           = openrtbKeyTmax
	KeyBidfloor       = openrtbKeyBidfloor
	KeyDevicetype     = openrtbKeyDevicetype
	KeyCat            = openrtbKeyCat
	KeyWseat          = openrtbKeyWseat
	KeyDeals          = openrtbKeyDeals
	KeyPmp            = openrtbKeyPmp
	KeyID             = openrtbKeyID
	KeySchain         = openrtbKeySchain
	KeySite           = openrtbKeySite
	KeyApp            = openrtbKeyApp
	KeyDOOH           = openrtbKeyDOOH
	KeyBanner         = openrtbKeyBanner
	KeyVideo          = openrtbKeyVideo
	KeyAudio          = openrtbKeyAudio
	KeyNative         = openrtbKeyNative
	KeyDevice         = openrtbKeyDevice
	KeyIP             = openrtbKeyIP
	KeyIPv6           = openrtbKeyIPv6
	KeyUA             = openrtbKeyUA
	KeyCountry        = openrtbKeyCountry
	KeyTest           = openrtbKeyTest
	KeyGDPR           = openrtbKeyGDPR
	KeyUSPrivacy      = openrtbKeyUSPrivacy
	KeyCur            = openrtbKeyCur
	KeyBidfloorcur    = openrtbKeyBidfloorcur
	KeyMaxduration    = openrtbKeyMaxduration
	KeyUser           = openrtbKeyUser
	KeyLanguage       = openrtbKeyLanguage
	KeyOS             = openrtbKeyOS
	KeyRegion         = openrtbKeyRegion
	KeySecure         = openrtbKeySecure
	KeyCoppa          = openrtbKeyCoppa
	KeyIFA            = openrtbKeyIFA
	KeyLMT            = openrtbKeyLMT
	KeyConnectiontype = openrtbKeyConnectiontype
	KeyBCat           = openrtbKeyBCat
	KeyBAdv           = openrtbKeyBAdv
	KeyBApp           = openrtbKeyBApp
	KeyBseat          = openrtbKeyBseat
	KeySource         = openrtbKeySource
)

func ScanOpenRTB26Payload(payload []byte) OpenRTB26Scan {
	return scanOpenRTB26Payload(payload)
}

func SectionWindow(payload []byte, start int, maxLen int) []byte {
	return sectionWindow(payload, start, maxLen)
}

func ScanDeviceWindow(win []byte) OpenRTB26DeviceScan {
	return scanDeviceWindow(win)
}

func ScanImpObject(obj []byte) OpenRTB26ImpObjScan {
	return scanImpObject(obj)
}

func ScanImpWindow(win []byte) OpenRTB26ImpWinScan {
	return scanImpWindow(win)
}

func ScanSupplyChainNodeObject(obj []byte) OpenRTB26SupplyChainNodeScan {
	return scanSupplyChainNodeObject(obj)
}

func ParseSupplyChainNodesAt(payload []byte, schainAt int) SupplyChainNodes {
	return parseSupplyChainNodesAt(payload, schainAt)
}

func (s OpenRTB26Scan) SecImp() int         { return s.sec.imp }
func (s OpenRTB26Scan) SecDevice() int      { return s.sec.device }
func (s OpenRTB26Scan) SecSite() int        { return s.sec.site }
func (s OpenRTB26Scan) SecApp() int         { return s.sec.app }
func (s OpenRTB26Scan) SecUser() int        { return s.sec.user }
func (s OpenRTB26Scan) SecSource() int      { return s.sec.source }
func (s OpenRTB26Scan) SecDOOH() int        { return s.sec.dooh }
func (s OpenRTB26Scan) IdxRequestID() int   { return s.idxRequestID }
func (s OpenRTB26Scan) IdxBseat() int       { return s.idxBseat }
func (s OpenRTB26Scan) IdxTmax() int        { return s.idxTmax }
func (s OpenRTB26Scan) IdxBidfloor() int    { return s.idxBidfloor }
func (s OpenRTB26Scan) IdxDevicetype() int  { return s.idxDevicetype }
func (s OpenRTB26Scan) IdxCat() int         { return s.idxCat }
func (s OpenRTB26Scan) IdxWseat() int       { return s.idxWseat }
func (s OpenRTB26Scan) IdxSchain() int      { return s.idxSchain }
func (s OpenRTB26Scan) IdxTest() int        { return s.idxTest }
func (s OpenRTB26Scan) IdxMaxduration() int { return s.idxMaxduration }
func (s OpenRTB26Scan) IdxCoppa() int       { return s.idxCoppa }
func (s OpenRTB26Scan) IdxGDPR() int        { return s.idxGDPR }
func (s OpenRTB26Scan) IdxUSPrivacy() int   { return s.idxUSPrivacy }
func (s OpenRTB26Scan) IdxCur() int         { return s.idxCur }
func (s OpenRTB26Scan) IdxBidfloorcur() int { return s.idxBidfloorcur }
func (s OpenRTB26Scan) IdxBCat() int        { return s.idxBCat }
func (s OpenRTB26Scan) IdxBAdv() int        { return s.idxBAdv }
func (s OpenRTB26Scan) IdxBApp() int        { return s.idxBApp }

func (s OpenRTB26DeviceScan) IdxIP() int             { return s.idxIP }
func (s OpenRTB26DeviceScan) IdxIPv6() int           { return s.idxIPv6 }
func (s OpenRTB26DeviceScan) IdxUA() int             { return s.idxUA }
func (s OpenRTB26DeviceScan) IdxCountry() int        { return s.idxCountry }
func (s OpenRTB26DeviceScan) IdxOS() int             { return s.idxOS }
func (s OpenRTB26DeviceScan) IdxLanguage() int       { return s.idxLanguage }
func (s OpenRTB26DeviceScan) IdxRegion() int         { return s.idxRegion }
func (s OpenRTB26DeviceScan) IdxIFA() int            { return s.idxIFA }
func (s OpenRTB26DeviceScan) IdxLMT() int            { return s.idxLMT }
func (s OpenRTB26DeviceScan) IdxConnectiontype() int { return s.idxConnectiontype }

func (s OpenRTB26ImpObjScan) IdxID() int          { return s.idxID }
func (s OpenRTB26ImpObjScan) IdxBidfloor() int    { return s.idxBidfloor }
func (s OpenRTB26ImpObjScan) IdxBanner() int      { return s.idxBanner }
func (s OpenRTB26ImpObjScan) IdxVideo() int       { return s.idxVideo }
func (s OpenRTB26ImpObjScan) IdxAudio() int       { return s.idxAudio }
func (s OpenRTB26ImpObjScan) IdxNative() int      { return s.idxNative }
func (s OpenRTB26ImpObjScan) IdxSecure() int      { return s.idxSecure }
func (s OpenRTB26ImpObjScan) IdxDealID() int      { return s.idxDealID }
func (s OpenRTB26ImpObjScan) IdxBannerW() int     { return s.idxBannerW }
func (s OpenRTB26ImpObjScan) IdxBannerH() int     { return s.idxBannerH }
func (s OpenRTB26ImpObjScan) IdxVideoW() int      { return s.idxVideoW }
func (s OpenRTB26ImpObjScan) IdxMaxduration() int { return s.idxMaxduration }

func (s OpenRTB26ImpWinScan) IdxBanner() int  { return s.idxBanner }
func (s OpenRTB26ImpWinScan) IdxVideo() int   { return s.idxVideo }
func (s OpenRTB26ImpWinScan) IdxSecure() int  { return s.idxSecure }
func (s OpenRTB26ImpWinScan) IdxPrivate() int { return s.idxPrivate }
func (s OpenRTB26ImpWinScan) IdxMetric() int  { return s.idxMetric }

func (s OpenRTB26SupplyChainNodeScan) IdxAsi() int { return s.idxAsi }
func (s OpenRTB26SupplyChainNodeScan) IdxSid() int { return s.idxSid }

const (
	OrtbDealIDMax      = ortbDealIDMax
	OrtbMaxDepth       = ortbMaxDepth
	OpenRTB26SeatIDMax = openrtb26SeatIDMax
)

func ExchangeReady(h *OpenRTB26Hot, cold *OpenRTB26Cold, cfg ExchangeConfig) bool {
	return exchangeReady(h, cold, cfg)
}

func ImpSlotExchangeReady(s *OpenRTB26ImpSlot) bool {
	return impSlotExchangeReady(s)
}

func BlockedCatMaskFromCold(cold *OpenRTB26Cold) uint64 {
	return blockedCatMaskFromCold(cold)
}

func CategoryBitFromIABCode(code []byte) uint64 {
	return categoryBitFromIABCode(code)
}

func ParseJSONIntField(payload []byte, start int) int64 {
	return parseJSONIntField(payload, start)
}

func ParseDecimalMicroField(payload []byte, start int) int64 {
	return parseDecimalMicroField(payload, start)
}

func ParseCategoryMaskFromArray(payload []byte, catIdx int) uint64 {
	return parseCategoryMaskFromArray(payload, catIdx)
}

func NormalizeCountryBytes(src []byte, dst []byte) int {
	return normalizeCountryBytes(src, dst)
}

func NormalizeRegionBytes(src []byte, dst []byte) int {
	return normalizeRegionBytes(src, dst)
}

func ParseImpObjectCountAt(payload []byte, impIdx int) int {
	return parseImpObjectCountAt(payload, impIdx)
}

func ParseSeatJSONArrayAt(payload []byte, start int, dst [][OpenRTB26SeatIDMax]byte, lens []uint8) uint8 {
	return parseSeatJSONArrayAt(payload, start, dst, lens)
}

func ForEachImpObject(payload []byte, impIdx int, fn func(obj []byte) bool) {
	foreachImpObject(payload, impIdx, fn)
}

func ParseImpSlot(obj []byte, slot *OpenRTB26ImpSlot) bool {
	return parseImpSlot(obj, slot)
}
