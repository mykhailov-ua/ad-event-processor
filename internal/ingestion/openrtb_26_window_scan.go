package ingestion

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

func (s *openrtb26ImpObjScan) initMiss() {
	s.idxID = ortbScanMiss
	s.idxBidfloor = ortbScanMiss
	s.idxBanner = ortbScanMiss
	s.idxVideo = ortbScanMiss
	s.idxAudio = ortbScanMiss
	s.idxNative = ortbScanMiss
	s.idxSecure = ortbScanMiss
	s.idxDeals = ortbScanMiss
	s.idxPmp = ortbScanMiss
	s.idxDealID = ortbScanMiss
	s.idxDealBidfloor = ortbScanMiss
	s.idxWseat = ortbScanMiss
	s.idxBannerW = ortbScanMiss
	s.idxBannerH = ortbScanMiss
	s.idxVideoW = ortbScanMiss
	s.idxVideoH = ortbScanMiss
	s.idxMaxduration = ortbScanMiss
}

func (s *openrtb26ImpObjScan) pmpSearchFrom() int {
	searchFrom := s.idxDeals
	if s.idxPmp >= 0 && (searchFrom < 0 || s.idxPmp < searchFrom) {
		return s.idxPmp
	}
	return searchFrom
}

func (s *openrtb26ImpObjScan) inBannerSpan(at int) bool {
	return s.idxBanner >= 0 && at > s.idxBanner && at < s.idxBanner+ortbWinBannerSpan
}

func (s *openrtb26ImpObjScan) inVideoSpan(at int) bool {
	return s.idxVideo >= 0 && at > s.idxVideo && at < s.idxVideo+ortbWinVideoSpan
}

func (s *openrtb26ImpObjScan) record(k ortb26WinKeyID, at int) {
	switch k {
	case ortbWinKeyID:
		if s.idxID < 0 {
			s.idxID = at
		}
		if searchFrom := s.pmpSearchFrom(); searchFrom >= 0 && at > searchFrom && s.idxDealID < 0 {
			s.idxDealID = at
		}
	case ortbWinKeyBidfloor:
		if s.idxBidfloor < 0 {
			s.idxBidfloor = at
		}
		if searchFrom := s.pmpSearchFrom(); searchFrom >= 0 && at > searchFrom && s.idxDealBidfloor < 0 {
			s.idxDealBidfloor = at
		}
	case ortbWinKeyBanner:
		if s.idxBanner < 0 {
			s.idxBanner = at
		}
	case ortbWinKeyVideo:
		if s.idxVideo < 0 {
			s.idxVideo = at
		}
	case ortbWinKeyAudio:
		if s.idxAudio < 0 {
			s.idxAudio = at
		}
	case ortbWinKeyNative:
		if s.idxNative < 0 {
			s.idxNative = at
		}
	case ortbWinKeySecure:
		if s.idxSecure < 0 {
			s.idxSecure = at
		}
	case ortbWinKeyDeals:
		if s.idxDeals < 0 {
			s.idxDeals = at
		}
	case ortbWinKeyPmp:
		if s.idxPmp < 0 {
			s.idxPmp = at
		}
	case ortbWinKeyWseat:
		if s.idxWseat < 0 {
			s.idxWseat = at
		}
	case ortbWinKeyW:
		if s.inBannerSpan(at) && s.idxBannerW < 0 {
			s.idxBannerW = at
		}
		if s.inVideoSpan(at) && s.idxVideoW < 0 {
			s.idxVideoW = at
		}
	case ortbWinKeyH:
		if s.inBannerSpan(at) && s.idxBannerH < 0 {
			s.idxBannerH = at
		}
		if s.inVideoSpan(at) && s.idxVideoH < 0 {
			s.idxVideoH = at
		}
	case ortbWinKeyMaxduration:
		if s.inVideoSpan(at) && s.idxMaxduration < 0 {
			s.idxMaxduration = at
		}
	}
}

func scanImpObject(obj []byte) openrtb26ImpObjScan {
	var s openrtb26ImpObjScan
	s.initMiss()
	scanOpenRTB26Window(obj, s.record)
	return s
}

type openrtb26ImpWinScan struct {
	idxBanner, idxVideo, idxAudio, idxNative, idxSecure int
	idxDeals, idxPmp, idxDealID, idxDealBidfloor        int
	idxBannerW, idxBannerH, idxVideoW, idxVideoH        int
	idxPrivate, idxMetric                               int
}

func (s *openrtb26ImpWinScan) initMiss() {
	s.idxBanner = ortbScanMiss
	s.idxVideo = ortbScanMiss
	s.idxAudio = ortbScanMiss
	s.idxNative = ortbScanMiss
	s.idxSecure = ortbScanMiss
	s.idxDeals = ortbScanMiss
	s.idxPmp = ortbScanMiss
	s.idxDealID = ortbScanMiss
	s.idxDealBidfloor = ortbScanMiss
	s.idxBannerW = ortbScanMiss
	s.idxBannerH = ortbScanMiss
	s.idxVideoW = ortbScanMiss
	s.idxVideoH = ortbScanMiss
	s.idxPrivate = ortbScanMiss
	s.idxMetric = ortbScanMiss
}

func (s *openrtb26ImpWinScan) pmpSearchFrom() int {
	searchFrom := s.idxDeals
	if s.idxPmp >= 0 && (searchFrom < 0 || s.idxPmp < searchFrom) {
		return s.idxPmp
	}
	return searchFrom
}

func (s *openrtb26ImpWinScan) inBannerSpan(at int) bool {
	return s.idxBanner >= 0 && at > s.idxBanner && at < s.idxBanner+ortbWinBannerSpan
}

func (s *openrtb26ImpWinScan) inVideoSpan(at int) bool {
	return s.idxVideo >= 0 && at > s.idxVideo && at < s.idxVideo+ortbWinVideoSpan
}

func (s *openrtb26ImpWinScan) record(k ortb26WinKeyID, at int) {
	switch k {
	case ortbWinKeyBanner:
		if s.idxBanner < 0 {
			s.idxBanner = at
		}
	case ortbWinKeyVideo:
		if s.idxVideo < 0 {
			s.idxVideo = at
		}
	case ortbWinKeyAudio:
		if s.idxAudio < 0 {
			s.idxAudio = at
		}
	case ortbWinKeyNative:
		if s.idxNative < 0 {
			s.idxNative = at
		}
	case ortbWinKeySecure:
		if s.idxSecure < 0 {
			s.idxSecure = at
		}
	case ortbWinKeyDeals:
		if s.idxDeals < 0 {
			s.idxDeals = at
		}
	case ortbWinKeyPmp:
		if s.idxPmp < 0 {
			s.idxPmp = at
		}
	case ortbWinKeyPrivate:
		if s.idxPrivate < 0 {
			s.idxPrivate = at
		}
	case ortbWinKeyMetric:
		if s.idxMetric < 0 {
			s.idxMetric = at
		}
	case ortbWinKeyID:
		if searchFrom := s.pmpSearchFrom(); searchFrom >= 0 && at > searchFrom && s.idxDealID < 0 {
			s.idxDealID = at
		}
	case ortbWinKeyBidfloor:
		if searchFrom := s.pmpSearchFrom(); searchFrom >= 0 && at > searchFrom && s.idxDealBidfloor < 0 {
			s.idxDealBidfloor = at
		}
	case ortbWinKeyW:
		if s.inBannerSpan(at) && s.idxBannerW < 0 {
			s.idxBannerW = at
		}
		if s.inVideoSpan(at) && s.idxVideoW < 0 {
			s.idxVideoW = at
		}
	case ortbWinKeyH:
		if s.inBannerSpan(at) && s.idxBannerH < 0 {
			s.idxBannerH = at
		}
		if s.inVideoSpan(at) && s.idxVideoH < 0 {
			s.idxVideoH = at
		}
	}
}

func scanImpWindow(win []byte) openrtb26ImpWinScan {
	var s openrtb26ImpWinScan
	s.initMiss()
	scanOpenRTB26Window(win, s.record)
	return s
}

type openrtb26SiteScan struct {
	idxDomain, idxPage int
}

func (s *openrtb26SiteScan) initMiss() {
	s.idxDomain = ortbScanMiss
	s.idxPage = ortbScanMiss
}

func (s *openrtb26SiteScan) record(k ortb26WinKeyID, at int) {
	switch k {
	case ortbWinKeyDomain:
		if s.idxDomain < 0 {
			s.idxDomain = at
		}
	case ortbWinKeyPage:
		if s.idxPage < 0 {
			s.idxPage = at
		}
	}
}

func scanSiteWindow(win []byte) openrtb26SiteScan {
	var s openrtb26SiteScan
	s.initMiss()
	scanOpenRTB26Window(win, s.record)
	return s
}

type openrtb26AppScan struct {
	idxBundle, idxVer int
}

func (s *openrtb26AppScan) initMiss() {
	s.idxBundle = ortbScanMiss
	s.idxVer = ortbScanMiss
}

func (s *openrtb26AppScan) record(k ortb26WinKeyID, at int) {
	switch k {
	case ortbWinKeyBundle:
		if s.idxBundle < 0 {
			s.idxBundle = at
		}
	case ortbWinKeyVer:
		if s.idxVer < 0 {
			s.idxVer = at
		}
	}
}

func scanAppWindow(win []byte) openrtb26AppScan {
	var s openrtb26AppScan
	s.initMiss()
	scanOpenRTB26Window(win, s.record)
	return s
}

type openrtb26ExtScan struct {
	idxTid, idxEids, idxEidSource, idxEidUID                  int
	idxMetric, idxMetricType, idxMetricVendor, idxMetricValue int
}

func (s *openrtb26ExtScan) initMiss() {
	s.idxTid = ortbScanMiss
	s.idxEids = ortbScanMiss
	s.idxEidSource = ortbScanMiss
	s.idxEidUID = ortbScanMiss
	s.idxMetric = ortbScanMiss
	s.idxMetricType = ortbScanMiss
	s.idxMetricVendor = ortbScanMiss
	s.idxMetricValue = ortbScanMiss
}

func (s *openrtb26ExtScan) inMetricSpan(at int) bool {
	return s.idxMetric >= 0 && at > s.idxMetric
}

func (s *openrtb26ExtScan) inEidsSpan(at int) bool {
	return s.idxEids >= 0 && at > s.idxEids
}

func (s *openrtb26ExtScan) record(k ortb26WinKeyID, at int) {
	switch k {
	case ortbWinKeyTid:
		if s.idxTid < 0 {
			s.idxTid = at
		}
	case ortbWinKeyEids:
		if s.idxEids < 0 {
			s.idxEids = at
		}
	case ortbWinKeyMetric:
		if s.idxMetric < 0 {
			s.idxMetric = at
		}
	case ortbWinKeySource:
		if s.inEidsSpan(at) && s.idxEidSource < 0 {
			s.idxEidSource = at
		}
	case ortbWinKeyUids:

	case ortbWinKeyID:
		if s.inEidsSpan(at) && s.idxEidUID < 0 {
			s.idxEidUID = at
		}
	case ortbWinKeyType:
		if s.inMetricSpan(at) && s.idxMetricType < 0 {
			s.idxMetricType = at
		}
	case ortbWinKeyVendor:
		if s.inMetricSpan(at) && s.idxMetricVendor < 0 {
			s.idxMetricVendor = at
		}
	case ortbWinKeyValue:
		if s.inMetricSpan(at) && s.idxMetricValue < 0 {
			s.idxMetricValue = at
		}
	}
}

func scanExtWindow(win []byte) openrtb26ExtScan {
	var s openrtb26ExtScan
	s.initMiss()
	scanOpenRTB26Window(win, s.record)
	return s
}

func ortbFieldAt(win []byte, idx int, key []byte) int {
	if idx < 0 {
		return -1
	}
	return idx + len(key)
}

type openrtb26SchainWinScan struct {
	idxNodes int
}

func (s *openrtb26SchainWinScan) initMiss() {
	s.idxNodes = ortbScanMiss
}

func (s *openrtb26SchainWinScan) record(k ortb26WinKeyID, at int) {
	if k == ortbWinKeyNodes && s.idxNodes < 0 {
		s.idxNodes = at
	}
}

func scanSchainWindow(win []byte) openrtb26SchainWinScan {
	var s openrtb26SchainWinScan
	s.initMiss()
	scanOpenRTB26Window(win, s.record)
	return s
}

type openrtb26SchainNodeScan struct {
	idxAsi, idxSid int
}

func (s *openrtb26SchainNodeScan) initMiss() {
	s.idxAsi = ortbScanMiss
	s.idxSid = ortbScanMiss
}

func (s *openrtb26SchainNodeScan) record(k ortb26WinKeyID, at int) {
	switch k {
	case ortbWinKeyAsi:
		if s.idxAsi < 0 {
			s.idxAsi = at
		}
	case ortbWinKeySid:
		if s.idxSid < 0 {
			s.idxSid = at
		}
	}
}

func scanSchainNodeObject(obj []byte) openrtb26SchainNodeScan {
	var s openrtb26SchainNodeScan
	s.initMiss()
	scanOpenRTB26Window(obj, s.record)
	return s
}
