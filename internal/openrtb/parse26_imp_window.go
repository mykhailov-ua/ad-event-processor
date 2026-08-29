package openrtb

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

type openrtb26SupplyChainNodeScan struct {
	idxAsi, idxSid int
}

func (s *openrtb26SupplyChainNodeScan) initMiss() {
	s.idxAsi = ortbScanMiss
	s.idxSid = ortbScanMiss
}

func (s *openrtb26SupplyChainNodeScan) record(k ortb26WinKeyID, at int) {
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

func scanSupplyChainNodeObject(obj []byte) openrtb26SupplyChainNodeScan {
	var s openrtb26SupplyChainNodeScan
	s.initMiss()
	scanOpenRTB26Window(obj, s.record)
	return s
}
