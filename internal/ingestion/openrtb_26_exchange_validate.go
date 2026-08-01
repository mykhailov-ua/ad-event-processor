package ingestion

const (
	openrtb26FlagSite uint64 = 1 << iota
	openrtb26FlagApp
	openrtb26FlagDOOH
	openrtb26FlagBanner
	openrtb26FlagVideo
	openrtb26FlagAudio
	openrtb26FlagNative
	openrtb26FlagDeviceIP
	openrtb26FlagDeviceIPv6
	openrtb26FlagDeviceUA
	openrtb26FlagGeoCountry
	openrtb26FlagTest
	openrtb26FlagGDPR
	openrtb26FlagUSPrivacyY
	openrtb26FlagEUR
	openrtb26FlagSecure
	openrtb26FlagCOPPA
	openrtb26FlagHasDomain
	openrtb26FlagHasBundle
	openrtb26FlagDeviceOS
	openrtb26FlagDeviceLang
	openrtb26FlagGeoRegion
	openrtb26FlagBuyerUID
	openrtb26FlagDeviceIFA
	openrtb26FlagDeviceLMT
	openrtb26FlagSourceTID
	openrtb26FlagPMPPrivate
	openrtb26FlagSitePage
	openrtb26FlagAppVer
	openrtb26FlagConnectionType
	openrtb26FlagEID
	openrtb26FlagMetric
	openrtb26FlagBCat
	openrtb26FlagBAdv
	openrtb26FlagBApp
	openrtb26FlagBSeat
)

func checkRegsPolicyParsed(h OpenRTB26Hot, policy string) bool {
	if policy == "off" || policy == "reject" {
		if policy == "reject" {
			if h.Flags&openrtb26FlagGDPR != 0 {
				return true
			}
			if h.Flags&openrtb26FlagUSPrivacyY != 0 {
				return true
			}
		}
	}
	return false
}

func checkCoppaPolicyParsed(h OpenRTB26Hot, policy string) bool {
	if policy != "reject" {
		return false
	}
	return h.Flags&openrtb26FlagCOPPA != 0
}
