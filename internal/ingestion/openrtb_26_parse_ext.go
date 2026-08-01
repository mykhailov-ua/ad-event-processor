package ingestion

import "bytes"

var (
	openrtbKeyTid     = []byte(`"tid"`)
	openrtbKeyPrivate = []byte(`"private"`)
	openrtbKeyPage    = []byte(`"page"`)
	openrtbKeyVer     = []byte(`"ver"`)
	openrtbKeyEids    = []byte(`"eids"`)
	openrtbKeyMetric  = []byte(`"metric"`)
	openrtbKeyType    = []byte(`"type"`)
	openrtbKeyValue   = []byte(`"value"`)
	openrtbKeyVendor  = []byte(`"vendor"`)
	openrtbKeyUids    = []byte(`"uids"`)
	openrtbKeySource  = []byte(`"source"`)
)

func parseExchangeExtensionFields(payload []byte, sec openrtb26Sections, impWin []byte, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	parseSourceTIDField(payload, sec.source, cold, hot)
	parsePMPPrivateField(impWin, hot)
	parseSitePageField(payload, sec.site, cold, hot)
	parseAppVerField(payload, sec.app, cold, hot)
	parseUserEIDFields(payload, sec.user, hot, cold)
	parseImpMetricFields(impWin, hot, cold)
}

func parseSourceTIDField(payload []byte, srcIdx int, cold *OpenRTB26Cold, hot *OpenRTB26Hot) {
	if srcIdx < 0 {
		return
	}
	window := sectionWindow(payload, srcIdx, 512)
	if tidIdx := bytes.Index(window, openrtbKeyTid); tidIdx >= 0 {
		cold.SourceTIDLen = uint8(parseQuotedField(window, tidIdx+len(openrtbKeyTid), cold.SourceTID[:]))
		if cold.SourceTIDLen > 0 {
			hot.Flags |= openrtb26FlagSourceTID
		}
	}
}

func parsePMPPrivateField(impWin []byte, hot *OpenRTB26Hot) {
	if len(impWin) == 0 {
		return
	}
	if privIdx := bytes.Index(impWin, openrtbKeyPrivate); privIdx >= 0 {
		if parseJSONIntField(impWin, privIdx+len(openrtbKeyPrivate)) == 1 {
			hot.PMPPrivate = 1
			hot.Flags |= openrtb26FlagPMPPrivate
		}
	}
}

func parseSitePageField(payload []byte, siteIdx int, cold *OpenRTB26Cold, hot *OpenRTB26Hot) {
	if siteIdx < 0 {
		return
	}
	window := sectionWindow(payload, siteIdx, 1024)
	if pageIdx := bytes.Index(window, openrtbKeyPage); pageIdx >= 0 {
		cold.SitePageLen = uint8(parseQuotedField(window, pageIdx+len(openrtbKeyPage), cold.SitePage[:]))
		if cold.SitePageLen > 0 {
			hot.Flags |= openrtb26FlagSitePage
		}
	}
}

func parseAppVerField(payload []byte, appIdx int, cold *OpenRTB26Cold, hot *OpenRTB26Hot) {
	if appIdx < 0 {
		return
	}
	window := sectionWindow(payload, appIdx, 512)
	if verIdx := bytes.Index(window, openrtbKeyVer); verIdx >= 0 {
		cold.AppVerLen = uint8(parseQuotedField(window, verIdx+len(openrtbKeyVer), cold.AppVer[:]))
		if cold.AppVerLen > 0 {
			hot.Flags |= openrtb26FlagAppVer
		}
	}
}

func parseUserEIDFields(payload []byte, userIdx int, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	if userIdx < 0 {
		return
	}
	window := sectionWindow(payload, userIdx, 4096)
	eidsIdx := bytes.Index(window, openrtbKeyEids)
	if eidsIdx < 0 {
		return
	}
	eidsSlice := window[eidsIdx:]
	hot.EIDCount = uint8(countJSONArrayObjects(eidsSlice))
	if srcIdx := bytes.Index(eidsSlice, openrtbKeySource); srcIdx >= 0 {
		cold.EIDSourceLen = uint8(parseQuotedField(eidsSlice, srcIdx+len(openrtbKeySource), cold.EIDSource[:]))
	}
	if uidsIdx := bytes.Index(eidsSlice, openrtbKeyUids); uidsIdx >= 0 {
		uidWin := eidsSlice[uidsIdx:]
		if idIdx := bytes.Index(uidWin, openrtbKeyID); idIdx >= 0 {
			cold.EIDUIDLen = uint8(parseQuotedField(uidWin, idIdx+len(openrtbKeyID), cold.EIDUID[:]))
		}
	}
	if cold.EIDSourceLen > 0 || cold.EIDUIDLen > 0 {
		hot.Flags |= openrtb26FlagEID
	}
}

func parseImpMetricFields(impWin []byte, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	if len(impWin) == 0 {
		return
	}
	metricIdx := bytes.Index(impWin, openrtbKeyMetric)
	if metricIdx < 0 {
		return
	}
	metricSlice := impWin[metricIdx:]
	objStart := bytes.IndexByte(metricSlice, '{')
	if objStart < 0 {
		return
	}
	obj := metricSlice[objStart:]
	if typeIdx := bytes.Index(obj, openrtbKeyType); typeIdx >= 0 {
		cold.MetricTypeLen = uint8(parseQuotedField(obj, typeIdx+len(openrtbKeyType), cold.MetricType[:]))
	}
	if vendorIdx := bytes.Index(obj, openrtbKeyVendor); vendorIdx >= 0 {
		cold.MetricVendorLen = uint8(parseQuotedField(obj, vendorIdx+len(openrtbKeyVendor), cold.MetricVendor[:]))
	}
	if valIdx := bytes.Index(obj, openrtbKeyValue); valIdx >= 0 {
		hot.MetricValuePPM = uint32(parseDecimalMicroField(obj, valIdx+len(openrtbKeyValue)))
	}
	if cold.MetricTypeLen > 0 || hot.MetricValuePPM > 0 {
		hot.Flags |= openrtb26FlagMetric
	}
}

func countJSONArrayObjects(slice []byte) int {
	i := bytes.IndexByte(slice, '[')
	if i < 0 {
		return 0
	}
	i++
	n := len(slice)
	if i >= n {
		return 0
	}
	_ = slice[n-1]
	count := 0
	depth := 0
	for i < n {
		switch slice[i] {
		case '{':
			if depth == 0 {
				count++
			}
			depth++
		case '}':
			depth--
		case ']':
			if depth == 0 {
				return count
			}
		}
		i++
	}
	return count
}
