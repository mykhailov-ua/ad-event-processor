package ingestion

import (
	"bytes"
)

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
)

func finalizeFcapUserHash(hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	if cold.UserIDLen > 0 {
		hot.FcapUserHash = hashUserIDBytes(cold.UserID[:cold.UserIDLen])
	} else if cold.BuyerUIDLen > 0 {
		hot.FcapUserHash = hashUserIDBytes(cold.BuyerUID[:cold.BuyerUIDLen])
	} else if cold.EIDUIDLen > 0 {
		hot.FcapUserHash = hashUserIDBytes(cold.EIDUID[:cold.EIDUIDLen])
	}
}

func ParseOpenRTB26(payload []byte) OpenRTB26Parsed {
	var out OpenRTB26Parsed
	ParseOpenRTB26Into(payload, &out)
	return out
}

func ParseOpenRTB26Into(payload []byte, out *OpenRTB26Parsed) {
	ParseOpenRTB26Split(payload, &out.OpenRTB26Hot, &out.OpenRTB26Cold)
}

func ParseOpenRTB26Split(payload []byte, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	*hot = OpenRTB26Hot{}
	*cold = OpenRTB26Cold{}
	n := len(payload)
	if n < 12 {
		return
	}
	sec := locateOpenRTB26Sections(payload)
	if sec.imp < 0 {
		return
	}
	hot.OK = true
	hot.DeviceType = 1
	hot.CategoryMask = 1
	hot.TmaxMs = 200
	hot.SeatCount = 1

	if sec.site >= 0 {
		hot.Flags |= openrtb26FlagSite
	}
	if sec.app >= 0 {
		hot.Flags |= openrtb26FlagApp
	}
	if sec.dooh >= 0 {
		hot.Flags |= openrtb26FlagDOOH
	}
	impWin := sectionWindow(payload, sec.imp, 1536)
	if bytes.Contains(impWin, openrtbKeyBanner) {
		hot.Flags |= openrtb26FlagBanner
	}
	if bytes.Contains(impWin, openrtbKeyVideo) {
		hot.Flags |= openrtb26FlagVideo
	}
	if bytes.Contains(impWin, openrtbKeyAudio) {
		hot.Flags |= openrtb26FlagAudio
	}
	if bytes.Contains(impWin, openrtbKeyNative) {
		hot.Flags |= openrtb26FlagNative
	}

	hot.RequestIDLen = parseRequestID(payload, sec.imp, hot.RequestID[:])
	hot.ImpIDLen = parseFirstImpIDAt(payload, sec.imp, hot.ImpID[:])
	hot.ImpCount = uint8(parseImpObjectCountAt(payload, sec.imp))

	if idx := bytes.Index(payload, openrtbKeyTmax); idx >= 0 {
		hot.TmaxMs = int32(parseJSONIntField(payload, idx+len(openrtbKeyTmax)))
		if hot.TmaxMs <= 0 {
			hot.TmaxMs = 200
		}
	}
	if idx := bytes.Index(payload, openrtbKeyBidfloor); idx >= 0 {
		hot.BidFloorMicro = parseDecimalMicroField(payload, idx+len(openrtbKeyBidfloor))
	}
	if idx := bytes.Index(payload, openrtbKeyDevicetype); idx >= 0 {
		hot.DeviceType = openrtbDeviceType(parseJSONIntField(payload, idx+len(openrtbKeyDevicetype)))
	}
	if idx := bytes.Index(payload, openrtbKeyCat); idx >= 0 {
		hot.CategoryMask = parseCategoryMaskFromArray(payload, idx)
	}
	if idx := bytes.Index(payload, openrtbKeyWseat); idx >= 0 {
		hot.SeatCount = parseSeatCountAt(payload, idx)
	}
	searchFrom := bytes.Index(impWin, openrtbKeyDeals)
	if searchFrom < 0 {
		searchFrom = bytes.Index(impWin, openrtbKeyPmp)
	}
	if searchFrom >= 0 {
		slice := impWin[searchFrom:]
		if idRel := bytes.Index(slice, openrtbKeyID); idRel >= 0 {
			hot.DealIDLen = uint8(parseQuotedField(impWin, searchFrom+idRel+len(openrtbKeyID), hot.DealID[:]))
		}
		if bfRel := bytes.Index(slice, openrtbKeyBidfloor); bfRel >= 0 {
			hot.DealBidFloorMicro = parseDecimalMicroField(slice, bfRel+len(openrtbKeyBidfloor))
		}
	}
	if idx := bytes.Index(payload, openrtbKeySchain); idx >= 0 {
		cold.Schain = parseSchainNodesAt(payload, idx)
	}
	parseDeviceSection(payload, sec, hot, cold)
	parseRegsFlags(payload, hot)
	parseCurrencyFlags(payload, hot)
	if idx := bytes.Index(payload, openrtbKeyTest); idx >= 0 {
		if parseJSONIntField(payload, idx+len(openrtbKeyTest)) == 1 {
			hot.Flags |= openrtb26FlagTest
		}
	}
	if idx := bytes.Index(payload, openrtbKeyMaxduration); idx >= 0 {
		dur := parseJSONIntField(payload, idx+len(openrtbKeyMaxduration))
		if dur > 0 {
			hot.MaxDurationSec = uint32(dur)
		}
	}
	cold.UserIDLen = parseUserIDAt(payload, sec.user, cold.UserID[:])
	parseInventoryFieldsAt(payload, sec, hot, cold)
	parseImpDimensionsAt(sec.imp, impWin, hot)
	if idx := bytes.Index(impWin, openrtbKeySecure); idx >= 0 {
		if parseJSONIntField(impWin, idx+len(openrtbKeySecure)) == 1 {
			hot.Flags |= openrtb26FlagSecure
		}
	}
	if idx := bytes.Index(payload, openrtbKeyCoppa); idx >= 0 {
		if parseJSONIntField(payload, idx+len(openrtbKeyCoppa)) == 1 {
			hot.Flags |= openrtb26FlagCOPPA
		}
	}
	parseExchangeExtensionFields(payload, sec, impWin, hot, cold)
	parseBlocklistFields(payload, hot, cold)
	parseSeatFields(payload, sec.imp, hot, cold)
	parseImpSlotsAt(payload, sec.imp, hot, cold)
	finalizeFcapUserHash(hot, cold)
}
