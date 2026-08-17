package ingestion

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
	ParseOpenRTB26Split(payload, &out.OpenRTB26Hot, &out.OpenRTB26Cold)
}

func ParseOpenRTB26Split(payload []byte, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	resetOpenRTB26Hot(hot)
	resetOpenRTB26Cold(cold)
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
		hot.Flags |= openrtb26FlagSite
	}
	if sec.app >= 0 {
		hot.Flags |= openrtb26FlagApp
	}
	if sec.dooh >= 0 {
		hot.Flags |= openrtb26FlagDOOH
	}
	impWin := sectionWindow(payload, sec.imp, 1536)
	iw := scanImpWindow(impWin)
	if iw.idxBanner >= 0 {
		hot.Flags |= openrtb26FlagBanner
	}
	if iw.idxVideo >= 0 {
		hot.Flags |= openrtb26FlagVideo
	}
	if iw.idxAudio >= 0 {
		hot.Flags |= openrtb26FlagAudio
	}
	if iw.idxNative >= 0 {
		hot.Flags |= openrtb26FlagNative
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
		hot.DeviceType = openrtbDeviceType(parseJSONIntField(payload, scan.idxDevicetype+len(openrtbKeyDevicetype)))
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
		cold.Schain = parseSchainNodesAt(payload, scan.idxSchain)
	}
	parseDeviceSection(payload, sec, hot, cold)
	parseRegsFlagsFromScan(payload, scan, hot)
	parseCurrencyFlagsFromScan(payload, scan, hot)
	if scan.idxTest >= 0 {
		if parseJSONIntField(payload, scan.idxTest+len(openrtbKeyTest)) == 1 {
			hot.Flags |= openrtb26FlagTest
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
			hot.Flags |= openrtb26FlagSecure
		}
	}
	if scan.idxCoppa >= 0 {
		if parseJSONIntField(payload, scan.idxCoppa+len(openrtbKeyCoppa)) == 1 {
			hot.Flags |= openrtb26FlagCOPPA
		}
	}
	parseExchangeExtensionFields(payload, sec, impWin, hot, cold)
	parseBlocklistFieldsFromScan(payload, scan, hot, cold)
	parseSeatFieldsFromScan(payload, scan, hot, cold)
	parseImpSlotsAt(payload, sec.imp, hot, cold)
	finalizeFcapUserHash(hot, cold)
}
