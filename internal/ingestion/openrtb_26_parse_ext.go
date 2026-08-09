package ingestion

//go:noinline
func parseExchangeExtensionFields(payload []byte, sec openrtb26Sections, impWin []byte, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	iw := scanImpWindow(impWin)
	parseSourceTIDField(payload, sec.source, cold, hot)
	parsePMPPrivateFieldFromScan(impWin, iw, hot)
	parseSitePageField(payload, sec.site, cold, hot)
	parseAppVerField(payload, sec.app, cold, hot)
	parseUserEIDFields(payload, sec.user, hot, cold)
	parseImpMetricFieldsFromScan(impWin, iw, hot, cold)
}

func parseSourceTIDField(payload []byte, srcIdx int, cold *OpenRTB26Cold, hot *OpenRTB26Hot) {
	if srcIdx < 0 {
		return
	}
	window := sectionWindow(payload, srcIdx, 512)
	ex := scanExtWindow(window)
	if ex.idxTid >= 0 {
		cold.SourceTIDLen = uint8(parseQuotedField(window, ortbFieldAt(window, ex.idxTid, openrtbKeyTid), cold.SourceTID[:]))
		if cold.SourceTIDLen > 0 {
			hot.Flags |= openrtb26FlagSourceTID
		}
	}
}

func parsePMPPrivateFieldFromScan(impWin []byte, iw openrtb26ImpWinScan, hot *OpenRTB26Hot) {
	if len(impWin) == 0 || iw.idxPrivate < 0 {
		return
	}
	if parseJSONIntField(impWin, ortbFieldAt(impWin, iw.idxPrivate, openrtbKeyPrivate)) == 1 {
		hot.PMPPrivate = 1
		hot.Flags |= openrtb26FlagPMPPrivate
	}
}

func parseSitePageField(payload []byte, siteIdx int, cold *OpenRTB26Cold, hot *OpenRTB26Hot) {
	if siteIdx < 0 {
		return
	}
	window := sectionWindow(payload, siteIdx, 1024)
	ss := scanSiteWindow(window)
	if ss.idxPage >= 0 {
		cold.SitePageLen = uint8(parseQuotedField(window, ortbFieldAt(window, ss.idxPage, openrtbKeyPage), cold.SitePage[:]))
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
	as := scanAppWindow(window)
	if as.idxVer >= 0 {
		cold.AppVerLen = uint8(parseQuotedField(window, ortbFieldAt(window, as.idxVer, openrtbKeyVer), cold.AppVer[:]))
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
	ex := scanExtWindow(window)
	if ex.idxEids < 0 {
		return
	}
	eidsSlice := window[ex.idxEids:]
	hot.EIDCount = uint8(countJSONArrayObjects(eidsSlice))
	if ex.idxEidSource >= 0 {
		cold.EIDSourceLen = uint8(parseQuotedField(window, ortbFieldAt(window, ex.idxEidSource, openrtbKeySource), cold.EIDSource[:]))
	}
	if ex.idxEidUID >= 0 {
		cold.EIDUIDLen = uint8(parseQuotedField(window, ortbFieldAt(window, ex.idxEidUID, openrtbKeyID), cold.EIDUID[:]))
	}
	if cold.EIDSourceLen > 0 || cold.EIDUIDLen > 0 {
		hot.Flags |= openrtb26FlagEID
	}
}

func parseImpMetricFieldsFromScan(impWin []byte, iw openrtb26ImpWinScan, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	if len(impWin) == 0 || iw.idxMetric < 0 {
		return
	}
	ex := scanExtWindow(impWin[iw.idxMetric:])
	if ex.idxMetricType >= 0 {
		cold.MetricTypeLen = uint8(parseQuotedField(impWin[iw.idxMetric:], ortbFieldAt(impWin[iw.idxMetric:], ex.idxMetricType, openrtbKeyType), cold.MetricType[:]))
	}
	if ex.idxMetricVendor >= 0 {
		cold.MetricVendorLen = uint8(parseQuotedField(impWin[iw.idxMetric:], ortbFieldAt(impWin[iw.idxMetric:], ex.idxMetricVendor, openrtbKeyVendor), cold.MetricVendor[:]))
	}
	if ex.idxMetricValue >= 0 {
		hot.MetricValuePPM = uint32(parseDecimalMicroField(impWin[iw.idxMetric:], ortbFieldAt(impWin[iw.idxMetric:], ex.idxMetricValue, openrtbKeyValue)))
	}
	if cold.MetricTypeLen > 0 || hot.MetricValuePPM > 0 {
		hot.Flags |= openrtb26FlagMetric
	}
}

func countJSONArrayObjects(slice []byte) int {
	i := -1
	for j, b := range slice {
		if b == '[' {
			i = j
			break
		}
	}
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
