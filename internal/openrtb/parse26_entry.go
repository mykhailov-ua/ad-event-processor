package openrtb

import (
	"bytes"
)

const (
	OpenRTB26FlagSite uint64 = 1 << iota
	OpenRTB26FlagApp
	OpenRTB26FlagDOOH
	OpenRTB26FlagBanner
	OpenRTB26FlagVideo
	OpenRTB26FlagAudio
	OpenRTB26FlagNative
	OpenRTB26FlagDeviceIP
	OpenRTB26FlagDeviceIPv6
	OpenRTB26FlagDeviceUA
	OpenRTB26FlagGeoCountry
	OpenRTB26FlagTest
	OpenRTB26FlagGDPR
	OpenRTB26FlagUSPrivacyY
	OpenRTB26FlagEUR
	OpenRTB26FlagSecure
	OpenRTB26FlagCOPPA
	OpenRTB26FlagHasDomain
	OpenRTB26FlagHasBundle
	OpenRTB26FlagDeviceOS
	OpenRTB26FlagDeviceLang
	OpenRTB26FlagGeoRegion
	OpenRTB26FlagBuyerUID
	OpenRTB26FlagDeviceIFA
	OpenRTB26FlagDeviceLMT
	OpenRTB26FlagSourceTID
	OpenRTB26FlagPMPPrivate
	OpenRTB26FlagSitePage
	OpenRTB26FlagAppVer
	OpenRTB26FlagConnectionType
	OpenRTB26FlagEID
	OpenRTB26FlagMetric
	OpenRTB26FlagBCat
	OpenRTB26FlagBAdv
	OpenRTB26FlagBApp
	OpenRTB26FlagBSeat
)

func CheckRegsPolicyParsed(h OpenRTB26Hot, policy string) bool {
	if policy == "off" || policy == "reject" {
		if policy == "reject" {
			if h.Flags&OpenRTB26FlagGDPR != 0 {
				return true
			}
			if h.Flags&OpenRTB26FlagUSPrivacyY != 0 {
				return true
			}
		}
	}
	return false
}

func CheckCoppaPolicyParsed(h OpenRTB26Hot, policy string) bool {
	if policy != "reject" {
		return false
	}
	return h.Flags&OpenRTB26FlagCOPPA != 0
}

const (
	openrtb26SeatMax       = 8
	openrtb26UserIDMax     = 128
	openrtb26GeoCountryMax = 3
	openrtb26DomainMax     = 128
	openrtb26BundleMax     = 128
	openrtb26OSMax         = 16
	openrtb26LangMax       = 8
	openrtb26BuyerUIDMax   = 128
	openrtb26RegionMax     = 3
)

var (
	openrtbKeyImp            = []byte(`"imp"`)
	openrtbKeyTmax           = []byte(`"tmax"`)
	openrtbKeyBidfloor       = []byte(`"bidfloor"`)
	openrtbKeyDevicetype     = []byte(`"devicetype"`)
	openrtbKeyCat            = []byte(`"cat"`)
	openrtbKeyWseat          = []byte(`"wseat"`)
	openrtbKeyDeals          = []byte(`"deals"`)
	openrtbKeyPmp            = []byte(`"pmp"`)
	openrtbKeyID             = []byte(`"id"`)
	openrtbKeySchain         = []byte(`"schain"`)
	openrtbKeyNodes          = []byte(`"nodes"`)
	openrtbKeyAsi            = []byte(`"asi"`)
	openrtbKeySid            = []byte(`"sid"`)
	openrtbKeySite           = []byte(`"site"`)
	openrtbKeyApp            = []byte(`"app"`)
	openrtbKeyDOOH           = []byte(`"dooh"`)
	openrtbKeyBanner         = []byte(`"banner"`)
	openrtbKeyVideo          = []byte(`"video"`)
	openrtbKeyAudio          = []byte(`"audio"`)
	openrtbKeyNative         = []byte(`"native"`)
	openrtbKeyDevice         = []byte(`"device"`)
	openrtbKeyIP             = []byte(`"ip"`)
	openrtbKeyIPv6           = []byte(`"ipv6"`)
	openrtbKeyUA             = []byte(`"ua"`)
	openrtbKeyCountry        = []byte(`"country"`)
	openrtbKeyTest           = []byte(`"test"`)
	openrtbKeyGDPR           = []byte(`"gdpr"`)
	openrtbKeyUSPrivacy      = []byte(`"us_privacy"`)
	openrtbKeyCur            = []byte(`"cur"`)
	openrtbKeyBidfloorcur    = []byte(`"bidfloorcur"`)
	openrtbKeyMaxduration    = []byte(`"maxduration"`)
	openrtbKeyUser           = []byte(`"user"`)
	openrtbKeyDomain         = []byte(`"domain"`)
	openrtbKeyBundle         = []byte(`"bundle"`)
	openrtbKeyLanguage       = []byte(`"language"`)
	openrtbKeyOS             = []byte(`"os"`)
	openrtbKeyRegion         = []byte(`"region"`)
	openrtbKeyBuyeruid       = []byte(`"buyeruid"`)
	openrtbKeySecure         = []byte(`"secure"`)
	openrtbKeyCoppa          = []byte(`"coppa"`)
	openrtbKeyW              = []byte(`"w"`)
	openrtbKeyH              = []byte(`"h"`)
	openrtbKeyIFA            = []byte(`"ifa"`)
	openrtbKeyLMT            = []byte(`"lmt"`)
	openrtbKeyConnectiontype = []byte(`"connectiontype"`)
	openrtbKeyTid            = []byte(`"tid"`)
	openrtbKeyPrivate        = []byte(`"private"`)
	openrtbKeyPage           = []byte(`"page"`)
	openrtbKeyVer            = []byte(`"ver"`)
	openrtbKeyEids           = []byte(`"eids"`)
	openrtbKeyMetric         = []byte(`"metric"`)
	openrtbKeyType           = []byte(`"type"`)
	openrtbKeyValue          = []byte(`"value"`)
	openrtbKeyVendor         = []byte(`"vendor"`)
	openrtbKeyUids           = []byte(`"uids"`)
	openrtbKeySource         = []byte(`"source"`)
)

func finalizeFcapUserHash(hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	switch {
	case cold.UserIDLen > 0:
		hot.FcapUserHash = hashUserIDBytes(cold.UserID[:cold.UserIDLen])
	case cold.BuyerUIDLen > 0:
		hot.FcapUserHash = hashUserIDBytes(cold.BuyerUID[:cold.BuyerUIDLen])
	case cold.EIDUIDLen > 0:
		hot.FcapUserHash = hashUserIDBytes(cold.EIDUID[:cold.EIDUIDLen])
	}
}

func ParseOpenRTB26(payload []byte) OpenRTB26Parsed {
	var out OpenRTB26Parsed
	ParseOpenRTB26Into(payload, &out)
	return out
}

func ParseOpenRTB26Into(payload []byte, out *OpenRTB26Parsed) {
	ParseOpenRTB26Parsed(payload, out)
}

// ParseOpenRTB26Parsed reuses out across requests (connContext); resetOpenRTB26Parsed clears prior imp slots.
func ParseOpenRTB26Parsed(payload []byte, out *OpenRTB26Parsed) {
	if out == nil {
		return
	}
	resetOpenRTB26Parsed(out)
	parseOpenRTB26Fields(payload, &out.OpenRTB26Hot, &out.OpenRTB26Cold)
}

func ParseOpenRTB26Split(payload []byte, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	if p, ok := openRTB26ParsedFromSplit(hot, cold); ok {
		resetOpenRTB26Parsed(p)
	} else {
		resetOpenRTB26Hot(hot)
		resetOpenRTB26Cold(cold)
	}
	parseOpenRTB26Fields(payload, hot, cold)
}

// parseOpenRTB26Fields: DFA scan only (no encoding/json). Defaults before field extract: DeviceType=1, TmaxMs=200, SeatCount=1.
func parseOpenRTB26Fields(payload []byte, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	n := len(payload)
	if n < 12 {
		return
	}
	scan := scanOpenRTB26Payload(payload)
	sec := scan.sec
	if sec.imp < 0 {
		return
	}
	hot.OK = true
	hot.DeviceType = 1
	hot.CategoryMask = 1
	hot.TmaxMs = 200
	hot.SeatCount = 1

	if sec.site >= 0 {
		hot.Flags |= OpenRTB26FlagSite
	}
	if sec.app >= 0 {
		hot.Flags |= OpenRTB26FlagApp
	}
	if sec.dooh >= 0 {
		hot.Flags |= OpenRTB26FlagDOOH
	}
	impWin := sectionWindow(payload, sec.imp, 1536)
	iw := scanImpWindow(impWin)
	if iw.idxBanner >= 0 {
		hot.Flags |= OpenRTB26FlagBanner
	}
	if iw.idxVideo >= 0 {
		hot.Flags |= OpenRTB26FlagVideo
	}
	if iw.idxAudio >= 0 {
		hot.Flags |= OpenRTB26FlagAudio
	}
	if iw.idxNative >= 0 {
		hot.Flags |= OpenRTB26FlagNative
	}

	if scan.idxRequestID >= 0 {
		hot.RequestIDLen = uint8(parseQuotedField(payload, scan.idxRequestID+len(openrtbKeyID), hot.RequestID[:]))
	}
	hot.ImpIDLen = parseFirstImpIDAt(payload, sec.imp, hot.ImpID[:])
	hot.ImpCount = uint8(parseImpObjectCountAt(payload, sec.imp))

	if scan.idxTmax >= 0 {
		hot.TmaxMs = int32(parseJSONIntField(payload, scan.idxTmax+len(openrtbKeyTmax)))
		if hot.TmaxMs <= 0 {
			hot.TmaxMs = 200
		}
	}
	if scan.idxBidfloor >= 0 {
		hot.BidFloorMicro = parseDecimalMicroField(payload, scan.idxBidfloor+len(openrtbKeyBidfloor))
	}
	if scan.idxDevicetype >= 0 {
		hot.DeviceType = OpenRTBDeviceType(parseJSONIntField(payload, scan.idxDevicetype+len(openrtbKeyDevicetype)))
	}
	if scan.idxCat >= 0 {
		hot.CategoryMask = parseCategoryMaskFromArray(payload, scan.idxCat)
	}
	if scan.idxWseat >= 0 {
		hot.SeatCount = parseSeatCountAt(payload, scan.idxWseat)
	}
	if iw.idxDealID >= 0 {
		hot.DealIDLen = uint8(parseQuotedField(impWin, ortbFieldAt(impWin, iw.idxDealID, openrtbKeyID), hot.DealID[:]))
	}
	if iw.idxDealBidfloor >= 0 {
		hot.DealBidFloorMicro = parseDecimalMicroField(impWin, ortbFieldAt(impWin, iw.idxDealBidfloor, openrtbKeyBidfloor))
	}
	if scan.idxSchain >= 0 {
		cold.Schain = parseSupplyChainNodesAt(payload, scan.idxSchain)
	}
	parseDeviceSection(payload, sec, hot, cold)
	parseRegsFlagsFromScan(payload, scan, hot)
	parseCurrencyFlagsFromScan(payload, scan, hot)
	if scan.idxTest >= 0 {
		if parseJSONIntField(payload, scan.idxTest+len(openrtbKeyTest)) == 1 {
			hot.Flags |= OpenRTB26FlagTest
		}
	}
	if scan.idxMaxduration >= 0 {
		dur := parseJSONIntField(payload, scan.idxMaxduration+len(openrtbKeyMaxduration))
		if dur > 0 {
			hot.MaxDurationSec = uint32(dur)
		}
	}
	cold.UserIDLen = parseUserIDAt(payload, sec.user, cold.UserID[:])
	parseInventoryFieldsAt(payload, sec, hot, cold)
	parseImpDimensionsFromScan(impWin, iw, hot)
	if iw.idxSecure >= 0 {
		if parseJSONIntField(impWin, ortbFieldAt(impWin, iw.idxSecure, openrtbKeySecure)) == 1 {
			hot.Flags |= OpenRTB26FlagSecure
		}
	}
	if scan.idxCoppa >= 0 {
		if parseJSONIntField(payload, scan.idxCoppa+len(openrtbKeyCoppa)) == 1 {
			hot.Flags |= OpenRTB26FlagCOPPA
		}
	}
	parseExchangeExtensionFields(payload, sec, impWin, hot, cold)
	parseBlocklistFieldsFromScan(payload, scan, hot, cold)
	parseSeatFieldsFromScan(payload, scan, hot, cold)
	parseImpSlotsAt(payload, sec.imp, hot, cold)
	finalizeFcapUserHash(hot, cold)
}

var (
	openrtbKeyBCat = []byte(`"bcat"`)
	openrtbKeyBAdv = []byte(`"badv"`)
	openrtbKeyBApp = []byte(`"bapp"`)
)

var rtbExchangeADomain = []byte("exchange.local")

func parseBlocklistFieldsFromScan(payload []byte, scan openrtb26Scan, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	if scan.idxBCat >= 0 {
		cold.BCatCount = parseStringJSONArrayBCat(payload, scan.idxBCat+len(openrtbKeyBCat), cold)
		if cold.BCatCount > 0 {
			hot.Flags |= OpenRTB26FlagBCat
		}
	}
	if scan.idxBAdv >= 0 {
		cold.BAdvCount = parseStringJSONArrayBAdv(payload, scan.idxBAdv+len(openrtbKeyBAdv), cold)
		if cold.BAdvCount > 0 {
			hot.Flags |= OpenRTB26FlagBAdv
		}
	}
	if scan.idxBApp >= 0 {
		cold.BAppCount = parseStringJSONArrayBApp(payload, scan.idxBApp+len(openrtbKeyBApp), cold)
		if cold.BAppCount > 0 {
			hot.Flags |= OpenRTB26FlagBApp
		}
	}
	if cold.BCatCount > 0 {
		cold.BCatMask = blockedCatMaskFromCold(cold)
	}
}

func parseStringJSONArrayBCat(payload []byte, start int, cold *OpenRTB26Cold) uint8 {
	return parseStringJSONArrayAt(payload, start, openrtb26BlocklistMax, openrtb26BCatItemMax, blocklistBCat, cold)
}

func parseStringJSONArrayBAdv(payload []byte, start int, cold *OpenRTB26Cold) uint8 {
	return parseStringJSONArrayAt(payload, start, openrtb26BlocklistMax, openrtb26BAdvItemMax, blocklistBAdv, cold)
}

func parseStringJSONArrayBApp(payload []byte, start int, cold *OpenRTB26Cold) uint8 {
	return parseStringJSONArrayAt(payload, start, openrtb26BlocklistMax, openrtb26BundleMax, blocklistBApp, cold)
}

type blocklistKind uint8

const (
	blocklistBCat blocklistKind = iota
	blocklistBAdv
	blocklistBApp
)

func parseStringJSONArrayAt(payload []byte, start int, maxItems, maxLen int, kind blocklistKind, cold *OpenRTB26Cold) uint8 {
	if start < 0 || start >= len(payload) || maxItems <= 0 || maxLen <= 0 || cold == nil {
		return 0
	}
	i := start
	n := len(payload)
	for i < n && payload[i] != '[' {
		i++
	}
	if i >= n {
		return 0
	}
	i++
	count := uint8(0)
	for i < n && int(count) < maxItems {
		for i < n && (payload[i] == ' ' || payload[i] == '\t' || payload[i] == '\n' || payload[i] == '\r' || payload[i] == ',') {
			i++
		}
		if i >= n || payload[i] == ']' {
			break
		}
		if payload[i] != '"' {
			break
		}
		fieldStart := i + 1
		i++
		for i < n && payload[i] != '"' {
			i++
		}
		if i >= n {
			break
		}
		ln := i - fieldStart
		if ln > 0 && ln <= maxLen {
			field := payload[fieldStart:i]
			switch kind {
			case blocklistBCat:
				copy(cold.BCat[count][:], field)
				cold.BCatLen[count] = uint8(ln)
			case blocklistBAdv:
				copy(cold.BAdv[count][:], field)
				cold.BAdvLen[count] = uint8(ln)
			case blocklistBApp:
				copy(cold.BApp[count][:], field)
				cold.BAppLen[count] = uint8(ln)
			}
			count++
		}
		i++
	}
	return count
}

func CheckBlocklistsParsed(hot OpenRTB26Hot, cold *OpenRTB26Cold, blocklistEnforce bool) bool {
	if !blocklistEnforce || cold == nil {
		return false
	}
	if cold.BAppCount > 0 && cold.AppBundleLen > 0 {
		bundle := cold.AppBundle[:cold.AppBundleLen]
		for i := range int(cold.BAppCount) {
			if cold.BAppLen[i] == 0 {
				continue
			}
			if bytes.EqualFold(cold.BApp[i][:cold.BAppLen[i]], bundle) {
				return true
			}
		}
	}
	if cold.BAdvCount > 0 {
		for i := range int(cold.BAdvCount) {
			if cold.BAdvLen[i] == 0 {
				continue
			}
			if bytes.EqualFold(cold.BAdv[i][:cold.BAdvLen[i]], rtbExchangeADomain) {
				return true
			}
		}
	}
	_ = hot.Flags & OpenRTB26FlagBCat
	return false
}

func blockedCatMaskFromCold(cold *OpenRTB26Cold) uint64 {
	if cold == nil || cold.BCatCount == 0 {
		return 0
	}
	var mask uint64
	for i := range int(cold.BCatCount) {
		if cold.BCatLen[i] == 0 {
			continue
		}
		mask |= categoryBitFromIABCode(cold.BCat[i][:cold.BCatLen[i]])
	}
	return mask
}

func categoryBitFromIABCode(code []byte) uint64 {
	if len(code) == 0 {
		return 0
	}
	var mask uint64
	for i := range code {
		c := code[i]
		if c >= '0' && c <= '9' {
			mask |= uint64(1) << uint64(c-'0')
		}
	}
	if mask == 0 {
		return 1
	}
	return mask
}

//go:noinline
func parseDeviceSection(payload []byte, sec openrtb26Sections, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	if sec.device < 0 {
		return
	}
	window := sectionWindow(payload, sec.device, 1024)
	ds := scanDeviceWindow(window)
	if ds.idxIP >= 0 {
		if parseQuotedField(window, ortbFieldAt(window, ds.idxIP, openrtbKeyIP), nil) > 0 {
			hot.Flags |= OpenRTB26FlagDeviceIP
		}
	}
	if ds.idxIPv6 >= 0 {
		if parseQuotedField(window, ortbFieldAt(window, ds.idxIPv6, openrtbKeyIPv6), nil) > 0 {
			hot.Flags |= OpenRTB26FlagDeviceIPv6
		}
	}
	if ds.idxUA >= 0 {
		if parseQuotedField(window, ortbFieldAt(window, ds.idxUA, openrtbKeyUA), nil) > 0 {
			hot.Flags |= OpenRTB26FlagDeviceUA
		}
	}
	if ds.idxCountry >= 0 {
		var buf [8]byte
		ln := parseQuotedField(window, ortbFieldAt(window, ds.idxCountry, openrtbKeyCountry), buf[:])
		if ln > 0 {
			norm := normalizeCountryBytes(buf[:ln], hot.GeoCountry[:])
			hot.GeoCountryLen = uint8(norm)
			if norm > 0 {
				hot.Flags |= OpenRTB26FlagGeoCountry
			}
		}
	}
	if ds.idxOS >= 0 {
		cold.DeviceOSLen = uint8(parseQuotedField(window, ortbFieldAt(window, ds.idxOS, openrtbKeyOS), cold.DeviceOS[:]))
		if cold.DeviceOSLen > 0 {
			hot.Flags |= OpenRTB26FlagDeviceOS
		}
	}
	if ds.idxLanguage >= 0 {
		cold.DeviceLangLen = uint8(parseQuotedField(window, ortbFieldAt(window, ds.idxLanguage, openrtbKeyLanguage), cold.DeviceLang[:]))
		if cold.DeviceLangLen > 0 {
			hot.Flags |= OpenRTB26FlagDeviceLang
		}
	}
	if ds.idxRegion >= 0 {
		var buf [8]byte
		if ln := parseQuotedField(window, ortbFieldAt(window, ds.idxRegion, openrtbKeyRegion), buf[:]); ln > 0 {
			norm := normalizeRegionBytes(buf[:ln], cold.GeoRegion[:])
			cold.GeoRegionLen = uint8(norm)
			if norm > 0 {
				hot.Flags |= OpenRTB26FlagGeoRegion
			}
		}
	}
	if ds.idxIFA >= 0 {
		cold.DeviceIFALen = uint8(parseQuotedField(window, ortbFieldAt(window, ds.idxIFA, openrtbKeyIFA), cold.DeviceIFA[:]))
		if cold.DeviceIFALen > 0 {
			hot.Flags |= OpenRTB26FlagDeviceIFA
		}
	}
	if ds.idxLMT >= 0 {
		if parseJSONIntField(window, ortbFieldAt(window, ds.idxLMT, openrtbKeyLMT)) == 1 {
			hot.DeviceLMT = 1
			hot.Flags |= OpenRTB26FlagDeviceLMT
		}
	}
	if ds.idxConnectiontype >= 0 {
		ct := parseJSONIntField(window, ortbFieldAt(window, ds.idxConnectiontype, openrtbKeyConnectiontype))
		if ct >= 0 && ct <= 255 {
			hot.ConnectionType = uint8(ct)
			hot.Flags |= OpenRTB26FlagConnectionType
		}
	}
	if sec.user >= 0 {
		userWin := sectionWindow(payload, sec.user, 384)
		_, buIdx := scanUserWindow(userWin)
		if buIdx >= 0 {
			cold.BuyerUIDLen = uint8(parseQuotedField(userWin, ortbFieldAt(userWin, buIdx, openrtbKeyBuyeruid), cold.BuyerUID[:]))
			if cold.BuyerUIDLen > 0 {
				hot.Flags |= OpenRTB26FlagBuyerUID
			}
		}
	}
}
