package ingest

import "ad-event-processor/internal/openrtb"

type (
	openrtb26Scan                = openrtb.OpenRTB26Scan
	openrtb26Sections            = openrtb.OpenRTB26Sections
	openrtb26DeviceScan          = openrtb.OpenRTB26DeviceScan
	openrtb26ImpObjScan          = openrtb.OpenRTB26ImpObjScan
	openrtb26ImpWinScan          = openrtb.OpenRTB26ImpWinScan
	openrtb26SupplyChainNodeScan = openrtb.OpenRTB26SupplyChainNodeScan
)

var (
	openrtbKeyImp            = openrtb.KeyImp
	openrtbKeyTmax           = openrtb.KeyTmax
	openrtbKeyBidfloor       = openrtb.KeyBidfloor
	openrtbKeyDevicetype     = openrtb.KeyDevicetype
	openrtbKeyCat            = openrtb.KeyCat
	openrtbKeyWseat          = openrtb.KeyWseat
	openrtbKeyDeals          = openrtb.KeyDeals
	openrtbKeyPmp            = openrtb.KeyPmp
	openrtbKeyID             = openrtb.KeyID
	openrtbKeySchain         = openrtb.KeySchain
	openrtbKeySite           = openrtb.KeySite
	openrtbKeyApp            = openrtb.KeyApp
	openrtbKeyDOOH           = openrtb.KeyDOOH
	openrtbKeyBanner         = openrtb.KeyBanner
	openrtbKeyVideo          = openrtb.KeyVideo
	openrtbKeyAudio          = openrtb.KeyAudio
	openrtbKeyNative         = openrtb.KeyNative
	openrtbKeyDevice         = openrtb.KeyDevice
	openrtbKeyIP             = openrtb.KeyIP
	openrtbKeyIPv6           = openrtb.KeyIPv6
	openrtbKeyUA             = openrtb.KeyUA
	openrtbKeyCountry        = openrtb.KeyCountry
	openrtbKeyTest           = openrtb.KeyTest
	openrtbKeyGDPR           = openrtb.KeyGDPR
	openrtbKeyUSPrivacy      = openrtb.KeyUSPrivacy
	openrtbKeyCur            = openrtb.KeyCur
	openrtbKeyBidfloorcur    = openrtb.KeyBidfloorcur
	openrtbKeyMaxduration    = openrtb.KeyMaxduration
	openrtbKeyUser           = openrtb.KeyUser
	openrtbKeyLanguage       = openrtb.KeyLanguage
	openrtbKeyOS             = openrtb.KeyOS
	openrtbKeyRegion         = openrtb.KeyRegion
	openrtbKeySecure         = openrtb.KeySecure
	openrtbKeyCoppa          = openrtb.KeyCoppa
	openrtbKeyIFA            = openrtb.KeyIFA
	openrtbKeyLMT            = openrtb.KeyLMT
	openrtbKeyConnectiontype = openrtb.KeyConnectiontype
	openrtbKeyBCat           = openrtb.KeyBCat
	openrtbKeyBAdv           = openrtb.KeyBAdv
	openrtbKeyBApp           = openrtb.KeyBApp
	openrtbKeyBseat          = openrtb.KeyBseat
	openrtbKeySource         = openrtb.KeySource

	ParseDealIDBytes = openrtb.ParseDealIDBytes
)

func parseDecimalMicro(b []byte) int64 {
	return openrtb.ParseDecimalMicro(b)
}

func ortbSlice(payload []byte, off int, ln uint8) []byte {
	return openrtb.OrtbSlice(payload, off, ln)
}

func scanOpenRTB26Payload(payload []byte) openrtb26Scan {
	return openrtb.ScanOpenRTB26Payload(payload)
}

func sectionWindow(payload []byte, start int, maxLen int) []byte {
	return openrtb.SectionWindow(payload, start, maxLen)
}

func scanDeviceWindow(win []byte) openrtb26DeviceScan {
	return openrtb.ScanDeviceWindow(win)
}

func scanImpObject(obj []byte) openrtb26ImpObjScan {
	return openrtb.ScanImpObject(obj)
}

func scanImpWindow(win []byte) openrtb26ImpWinScan {
	return openrtb.ScanImpWindow(win)
}

func scanSchainNodeObject(obj []byte) openrtb26SupplyChainNodeScan {
	return openrtb.ScanSupplyChainNodeObject(obj)
}

func parseSchainNodesAt(payload []byte, schainAt int) openrtb.SupplyChainNodes {
	return openrtb.ParseSupplyChainNodesAt(payload, schainAt)
}

var (
	exchangeReady              = openrtb.ExchangeReady
	impSlotExchangeReady       = openrtb.ImpSlotExchangeReady
	blockedCatMaskFromCold     = openrtb.BlockedCatMaskFromCold
	categoryBitFromIABCode     = openrtb.CategoryBitFromIABCode
	parseJSONIntField          = openrtb.ParseJSONIntField
	parseDecimalMicroField     = openrtb.ParseDecimalMicroField
	parseCategoryMaskFromArray = openrtb.ParseCategoryMaskFromArray
	normalizeCountryBytes      = openrtb.NormalizeCountryBytes
	normalizeRegionBytes       = openrtb.NormalizeRegionBytes
	parseImpObjectCountAt      = openrtb.ParseImpObjectCountAt
	parseSeatJSONArrayAt       = openrtb.ParseSeatJSONArrayAt
	foreachImpObject           = openrtb.ForEachImpObject
	parseImpSlot               = openrtb.ParseImpSlot
	ParseDealID                = openrtb.ParseDealID
)

const (
	ortbDealIDMax = openrtb.OrtbDealIDMax
	ortbMaxDepth  = openrtb.OrtbMaxDepth
)
