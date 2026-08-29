package openrtb

type OpenRTB26Cold struct {
	UserID          [openrtb26UserIDMax]byte
	UserIDLen       uint8
	GeoRegion       [openrtb26RegionMax]byte
	GeoRegionLen    uint8
	SiteDomain      [openrtb26DomainMax]byte
	SiteDomainLen   uint8
	AppBundle       [openrtb26BundleMax]byte
	AppBundleLen    uint8
	DeviceOS        [openrtb26OSMax]byte
	DeviceOSLen     uint8
	DeviceLang      [openrtb26LangMax]byte
	DeviceLangLen   uint8
	BuyerUID        [openrtb26BuyerUIDMax]byte
	BuyerUIDLen     uint8
	DeviceIFA       [openrtb26IFAMax]byte
	DeviceIFALen    uint8
	SourceTID       [openrtb26SourceTIDMax]byte
	SourceTIDLen    uint8
	SitePage        [openrtb26SitePageMax]byte
	SitePageLen     uint8
	AppVer          [openrtb26AppVerMax]byte
	AppVerLen       uint8
	EIDSource       [openrtb26EIDSourceMax]byte
	EIDSourceLen    uint8
	EIDUID          [openrtb26EIDUIDMax]byte
	EIDUIDLen       uint8
	MetricType      [openrtb26MetricTypeMax]byte
	MetricTypeLen   uint8
	MetricVendor    [openrtb26MetricVendorMax]byte
	MetricVendorLen uint8
	BCat            [openrtb26BlocklistMax][openrtb26BCatItemMax]byte
	BCatLen         [openrtb26BlocklistMax]uint8
	BCatCount       uint8
	BAdv            [openrtb26BlocklistMax][openrtb26BAdvItemMax]byte
	BAdvLen         [openrtb26BlocklistMax]uint8
	BAdvCount       uint8
	BApp            [openrtb26BlocklistMax][openrtb26BundleMax]byte
	BAppLen         [openrtb26BlocklistMax]uint8
	BAppCount       uint8
	BCatMask        uint64
	BSeat           [openrtb26SeatMax][openrtb26SeatIDMax]byte
	BSeatLen        [openrtb26SeatMax]uint8
	BSeatCount      uint8
	Imps            [OpenRTB26ImpMax]OpenRTB26ImpSlot
	ImpSlots        uint8
	Schain          SupplyChainNodes
}

type OpenRTB26Parsed struct {
	OpenRTB26Hot
	OpenRTB26Cold
}

func (p OpenRTB26Parsed) ExchangeReady(cfg ExchangeConfig) bool {
	return exchangeReady(&p.OpenRTB26Hot, &p.OpenRTB26Cold, cfg)
}

func (h OpenRTB26Hot) ExchangeReady(cfg ExchangeConfig) bool {
	return exchangeReady(&h, nil, cfg)
}

func exchangeReady(h *OpenRTB26Hot, cold *OpenRTB26Cold, cfg ExchangeConfig) bool {
	if h == nil || !h.OK || h.RequestIDLen == 0 || h.ImpCount == 0 {
		return false
	}
	if cfg.MultiImpMax > 0 && int(h.ImpCount) > cfg.MultiImpMax {
		return false
	}
	if int(h.ImpCount) > OpenRTB26ImpMax {
		return false
	}
	inv := 0
	if h.Flags&OpenRTB26FlagSite != 0 {
		inv++
	}
	if h.Flags&OpenRTB26FlagApp != 0 {
		inv++
	}
	if h.Flags&OpenRTB26FlagDOOH != 0 {
		return false
	}
	if inv != 1 {
		return false
	}
	hasIP := h.Flags&(OpenRTB26FlagDeviceIP|OpenRTB26FlagDeviceIPv6) != 0
	if !hasIP || h.Flags&OpenRTB26FlagDeviceUA == 0 {
		return false
	}
	if cold != nil && cold.ImpSlots > 0 {
		if int(cold.ImpSlots) < int(h.ImpCount) {
			return false
		}
		for i := range int(h.ImpCount) {
			if !impSlotExchangeReady(&cold.Imps[i]) {
				return false
			}
		}
		return true
	}
	if h.ImpIDLen == 0 {
		return false
	}
	if h.Flags&(OpenRTB26FlagBanner|OpenRTB26FlagVideo) == 0 {
		return false
	}
	if h.Flags&(OpenRTB26FlagAudio|OpenRTB26FlagNative) != 0 {
		return false
	}
	if h.BidFloorMicro < 0 || h.DealBidFloorMicro < 0 {
		return false
	}
	return true
}

func impSlotExchangeReady(s *OpenRTB26ImpSlot) bool {
	if s == nil || s.ImpIDLen == 0 {
		return false
	}
	if s.Flags&(ImpSlotFlagBanner|ImpSlotFlagVideo) == 0 {
		return false
	}
	if s.Flags&(ImpSlotFlagAudio|ImpSlotFlagNative) != 0 {
		return false
	}
	if s.BidFloorMicro < 0 || s.DealBidFloorMicro < 0 {
		return false
	}
	return true
}

type ortb26WinKeyID uint8

const (
	ortbWinKeyNone ortb26WinKeyID = iota
	ortbWinKeyID
	ortbWinKeyIP
	ortbWinKeyIPv6
	ortbWinKeyUA
	ortbWinKeyCountry
	ortbWinKeyOS
	ortbWinKeyLanguage
	ortbWinKeyRegion
	ortbWinKeyIFA
	ortbWinKeyLMT
	ortbWinKeyConnectiontype
	ortbWinKeyBuyeruid
	ortbWinKeyBidfloor
	ortbWinKeyBanner
	ortbWinKeyVideo
	ortbWinKeyAudio
	ortbWinKeyNative
	ortbWinKeySecure
	ortbWinKeyDeals
	ortbWinKeyPmp
	ortbWinKeyPrivate
	ortbWinKeyMetric
	ortbWinKeyW
	ortbWinKeyH
	ortbWinKeyMaxduration
	ortbWinKeyWseat
	ortbWinKeyDomain
	ortbWinKeyBundle
	ortbWinKeyPage
	ortbWinKeyVer
	ortbWinKeyTid
	ortbWinKeyEids
	ortbWinKeySource
	ortbWinKeyUids
	ortbWinKeyType
	ortbWinKeyValue
	ortbWinKeyVendor
	ortbWinKeyNodes
	ortbWinKeyAsi
	ortbWinKeySid
)

const (
	ortbWinBannerSpan = 160
	ortbWinVideoSpan  = 200
)

func matchOpenRTB26WindowKey(b []byte, i, n int) ortb26WinKeyID {
	if !isOpenRTB26JSONKeyStart(b, i) {
		return ortbWinKeyNone
	}
	rem := n - i
	if rem < 4 {
		return ortbWinKeyNone
	}
	switch b[i+1] {
	case 'i':
		if rem >= len(openrtbKeyIPv6) && matchQuotedKeyAt(b, i, n, openrtbKeyIPv6) {
			return ortbWinKeyIPv6
		}
		if rem >= len(openrtbKeyIP) && matchQuotedKeyAt(b, i, n, openrtbKeyIP) {
			return ortbWinKeyIP
		}
		if rem >= len(openrtbKeyID) && matchQuotedKeyAt(b, i, n, openrtbKeyID) {
			return ortbWinKeyID
		}
		if rem >= len(openrtbKeyIFA) && matchQuotedKeyAt(b, i, n, openrtbKeyIFA) {
			return ortbWinKeyIFA
		}
	case 'u':
		if rem >= len(openrtbKeyUA) && matchQuotedKeyAt(b, i, n, openrtbKeyUA) {
			return ortbWinKeyUA
		}
		if rem >= len(openrtbKeyUids) && matchQuotedKeyAt(b, i, n, openrtbKeyUids) {
			return ortbWinKeyUids
		}
	case 'c':
		if rem >= len(openrtbKeyConnectiontype) && matchQuotedKeyAt(b, i, n, openrtbKeyConnectiontype) {
			return ortbWinKeyConnectiontype
		}
		if rem >= len(openrtbKeyCountry) && matchQuotedKeyAt(b, i, n, openrtbKeyCountry) {
			return ortbWinKeyCountry
		}
	case 'o':
		if rem >= len(openrtbKeyOS) && matchQuotedKeyAt(b, i, n, openrtbKeyOS) {
			return ortbWinKeyOS
		}
	case 'l':
		if rem >= len(openrtbKeyLanguage) && matchQuotedKeyAt(b, i, n, openrtbKeyLanguage) {
			return ortbWinKeyLanguage
		}
		if rem >= len(openrtbKeyLMT) && matchQuotedKeyAt(b, i, n, openrtbKeyLMT) {
			return ortbWinKeyLMT
		}
	case 'r':
		if rem >= len(openrtbKeyRegion) && matchQuotedKeyAt(b, i, n, openrtbKeyRegion) {
			return ortbWinKeyRegion
		}
	case 'b':
		if rem >= len(openrtbKeyBidfloor) && matchQuotedKeyAt(b, i, n, openrtbKeyBidfloor) {
			return ortbWinKeyBidfloor
		}
		if rem >= len(openrtbKeyBanner) && matchQuotedKeyAt(b, i, n, openrtbKeyBanner) {
			return ortbWinKeyBanner
		}
		if rem >= len(openrtbKeyBundle) && matchQuotedKeyAt(b, i, n, openrtbKeyBundle) {
			return ortbWinKeyBundle
		}
		if rem >= len(openrtbKeyBuyeruid) && matchQuotedKeyAt(b, i, n, openrtbKeyBuyeruid) {
			return ortbWinKeyBuyeruid
		}
	case 'v':
		if rem >= len(openrtbKeyVideo) && matchQuotedKeyAt(b, i, n, openrtbKeyVideo) {
			return ortbWinKeyVideo
		}
		if rem >= len(openrtbKeyVer) && matchQuotedKeyAt(b, i, n, openrtbKeyVer) {
			return ortbWinKeyVer
		}
		if rem >= len(openrtbKeyVendor) && matchQuotedKeyAt(b, i, n, openrtbKeyVendor) {
			return ortbWinKeyVendor
		}
		if rem >= len(openrtbKeyValue) && matchQuotedKeyAt(b, i, n, openrtbKeyValue) {
			return ortbWinKeyValue
		}
	case 'a':
		if rem >= len(openrtbKeyAsi) && matchQuotedKeyAt(b, i, n, openrtbKeyAsi) {
			return ortbWinKeyAsi
		}
		if rem >= len(openrtbKeyAudio) && matchQuotedKeyAt(b, i, n, openrtbKeyAudio) {
			return ortbWinKeyAudio
		}
	case 'n':
		if rem >= len(openrtbKeyNodes) && matchQuotedKeyAt(b, i, n, openrtbKeyNodes) {
			return ortbWinKeyNodes
		}
		if rem >= len(openrtbKeyNative) && matchQuotedKeyAt(b, i, n, openrtbKeyNative) {
			return ortbWinKeyNative
		}
	case 's':
		if rem >= len(openrtbKeySecure) && matchQuotedKeyAt(b, i, n, openrtbKeySecure) {
			return ortbWinKeySecure
		}
		if rem >= len(openrtbKeySid) && matchQuotedKeyAt(b, i, n, openrtbKeySid) {
			return ortbWinKeySid
		}
		if rem >= len(openrtbKeySource) && matchQuotedKeyAt(b, i, n, openrtbKeySource) {
			return ortbWinKeySource
		}
	case 'd':
		if rem >= len(openrtbKeyDeals) && matchQuotedKeyAt(b, i, n, openrtbKeyDeals) {
			return ortbWinKeyDeals
		}
		if rem >= len(openrtbKeyDomain) && matchQuotedKeyAt(b, i, n, openrtbKeyDomain) {
			return ortbWinKeyDomain
		}
	case 'p':
		if rem >= len(openrtbKeyPrivate) && matchQuotedKeyAt(b, i, n, openrtbKeyPrivate) {
			return ortbWinKeyPrivate
		}
		if rem >= len(openrtbKeyPmp) && matchQuotedKeyAt(b, i, n, openrtbKeyPmp) {
			return ortbWinKeyPmp
		}
		if rem >= len(openrtbKeyPage) && matchQuotedKeyAt(b, i, n, openrtbKeyPage) {
			return ortbWinKeyPage
		}
	case 'm':
		if rem >= len(openrtbKeyMaxduration) && matchQuotedKeyAt(b, i, n, openrtbKeyMaxduration) {
			return ortbWinKeyMaxduration
		}
		if rem >= len(openrtbKeyMetric) && matchQuotedKeyAt(b, i, n, openrtbKeyMetric) {
			return ortbWinKeyMetric
		}
	case 'w':
		if rem >= len(openrtbKeyWseat) && matchQuotedKeyAt(b, i, n, openrtbKeyWseat) {
			return ortbWinKeyWseat
		}
		if rem >= 3 && b[i+1] == 'w' && b[i+2] == '"' {
			return ortbWinKeyW
		}
	case 'h':
		if rem >= 3 && b[i+1] == 'h' && b[i+2] == '"' {
			return ortbWinKeyH
		}
	case 't':
		if rem >= len(openrtbKeyTid) && matchQuotedKeyAt(b, i, n, openrtbKeyTid) {
			return ortbWinKeyTid
		}
		if rem >= len(openrtbKeyType) && matchQuotedKeyAt(b, i, n, openrtbKeyType) {
			return ortbWinKeyType
		}
	case 'e':
		if rem >= len(openrtbKeyEids) && matchQuotedKeyAt(b, i, n, openrtbKeyEids) {
			return ortbWinKeyEids
		}
	}
	return ortbWinKeyNone
}

func scanOpenRTB26Window(win []byte, record func(ortb26WinKeyID, int)) {
	if record == nil || len(win) < 4 {
		return
	}
	n := len(win)
	_ = win[n-1]
	for i := range n {
		if win[i] != '"' {
			continue
		}
		k := matchOpenRTB26WindowKey(win, i, n)
		if k == ortbWinKeyNone {
			continue
		}
		record(k, i)
	}
}

type openrtb26DeviceScan struct {
	idxIP, idxIPv6, idxUA, idxCountry, idxOS, idxLanguage, idxRegion int
	idxIFA, idxLMT, idxConnectiontype, idxBuyeruid                   int
}

func (s *openrtb26DeviceScan) initMiss() {
	s.idxIP = ortbScanMiss
	s.idxIPv6 = ortbScanMiss
	s.idxUA = ortbScanMiss
	s.idxCountry = ortbScanMiss
	s.idxOS = ortbScanMiss
	s.idxLanguage = ortbScanMiss
	s.idxRegion = ortbScanMiss
	s.idxIFA = ortbScanMiss
	s.idxLMT = ortbScanMiss
	s.idxConnectiontype = ortbScanMiss
	s.idxBuyeruid = ortbScanMiss
}

func (s *openrtb26DeviceScan) record(k ortb26WinKeyID, at int) {
	switch k {
	case ortbWinKeyIP:
		if s.idxIP < 0 {
			s.idxIP = at
		}
	case ortbWinKeyIPv6:
		if s.idxIPv6 < 0 {
			s.idxIPv6 = at
		}
	case ortbWinKeyUA:
		if s.idxUA < 0 {
			s.idxUA = at
		}
	case ortbWinKeyCountry:
		if s.idxCountry < 0 {
			s.idxCountry = at
		}
	case ortbWinKeyOS:
		if s.idxOS < 0 {
			s.idxOS = at
		}
	case ortbWinKeyLanguage:
		if s.idxLanguage < 0 {
			s.idxLanguage = at
		}
	case ortbWinKeyRegion:
		if s.idxRegion < 0 {
			s.idxRegion = at
		}
	case ortbWinKeyIFA:
		if s.idxIFA < 0 {
			s.idxIFA = at
		}
	case ortbWinKeyLMT:
		if s.idxLMT < 0 {
			s.idxLMT = at
		}
	case ortbWinKeyConnectiontype:
		if s.idxConnectiontype < 0 {
			s.idxConnectiontype = at
		}
	case ortbWinKeyBuyeruid:
		if s.idxBuyeruid < 0 {
			s.idxBuyeruid = at
		}
	}
}

func scanDeviceWindow(win []byte) openrtb26DeviceScan {
	var s openrtb26DeviceScan
	s.initMiss()
	scanOpenRTB26Window(win, s.record)
	return s
}

func scanUserWindow(win []byte) (idxID, idxBuyeruid int) {
	idxID = ortbScanMiss
	idxBuyeruid = ortbScanMiss
	scanOpenRTB26Window(win, func(k ortb26WinKeyID, at int) {
		switch k {
		case ortbWinKeyID:
			if idxID < 0 {
				idxID = at
			}
		case ortbWinKeyBuyeruid:
			if idxBuyeruid < 0 {
				idxBuyeruid = at
			}
		}
	})
	return idxID, idxBuyeruid
}

type openrtb26ImpObjScan struct {
	idxID, idxBidfloor, idxBanner, idxVideo, idxAudio, idxNative, idxSecure int
	idxDeals, idxPmp, idxDealID, idxDealBidfloor, idxWseat                  int
	idxBannerW, idxBannerH, idxVideoW, idxVideoH, idxMaxduration            int
}
