package openrtb

import (
	"bytes"
	"time"
)

func parseRegsFlagsFromScan(payload []byte, scan openrtb26Scan, hot *OpenRTB26Hot) {
	if scan.idxGDPR >= 0 {
		if parseJSONIntField(payload, scan.idxGDPR+len(openrtbKeyGDPR)) == 1 {
			hot.Flags |= OpenRTB26FlagGDPR
		}
	}
	if scan.idxUSPrivacy >= 0 {
		var buf [16]byte
		if ln := parseQuotedField(payload, scan.idxUSPrivacy+len(openrtbKeyUSPrivacy), buf[:]); ln > 0 && buf[0] == 'Y' {
			hot.Flags |= OpenRTB26FlagUSPrivacyY
		}
	}
}

func parseCurrencyFlagsFromScan(payload []byte, scan openrtb26Scan, hot *OpenRTB26Hot) {
	if scan.idxCur >= 0 {
		var buf [8]byte
		if ln := parseQuotedField(payload, scan.idxCur+len(openrtbKeyCur), buf[:]); ln >= 3 {
			if buf[0] == 'E' && buf[1] == 'U' && buf[2] == 'R' {
				hot.Flags |= OpenRTB26FlagEUR
			}
		}
	}
	if hot.Flags&OpenRTB26FlagEUR == 0 && scan.idxBidfloorcur >= 0 {
		var buf [8]byte
		if ln := parseQuotedField(payload, scan.idxBidfloorcur+len(openrtbKeyBidfloorcur), buf[:]); ln >= 3 {
			if buf[0] == 'E' && buf[1] == 'U' && buf[2] == 'R' {
				hot.Flags |= OpenRTB26FlagEUR
			}
		}
	}
}

func parseUserIDAt(payload []byte, userIdx int, dst []byte) uint8 {
	if userIdx < 0 {
		return 0
	}
	slice := sectionWindow(payload, userIdx, 384)
	idIdx, _ := scanUserWindow(slice)
	if idIdx >= 0 {
		return uint8(parseQuotedField(slice, ortbFieldAt(slice, idIdx, openrtbKeyID), dst))
	}
	return 0
}

func parseInventoryFieldsAt(payload []byte, sec openrtb26Sections, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	if sec.site >= 0 {
		window := sectionWindow(payload, sec.site, 640)
		ss := scanSiteWindow(window)
		if ss.idxDomain >= 0 {
			cold.SiteDomainLen = uint8(parseQuotedField(window, ortbFieldAt(window, ss.idxDomain, openrtbKeyDomain), cold.SiteDomain[:]))
			if cold.SiteDomainLen > 0 {
				hot.Flags |= OpenRTB26FlagHasDomain
			}
		}
	}
	if sec.app >= 0 {
		window := sectionWindow(payload, sec.app, 640)
		as := scanAppWindow(window)
		if as.idxBundle >= 0 {
			cold.AppBundleLen = uint8(parseQuotedField(window, ortbFieldAt(window, as.idxBundle, openrtbKeyBundle), cold.AppBundle[:]))
			if cold.AppBundleLen > 0 {
				hot.Flags |= OpenRTB26FlagHasBundle
			}
		}
	}
}

func parseImpDimensionsFromScan(window []byte, iw openrtb26ImpWinScan, hot *OpenRTB26Hot) {
	if len(window) == 0 || hot == nil {
		return
	}
	if iw.idxBannerW >= 0 {
		hot.BannerW = uint16(parseJSONIntField(window, ortbFieldAt(window, iw.idxBannerW, openrtbKeyW)))
	}
	if iw.idxBannerH >= 0 {
		hot.BannerH = uint16(parseJSONIntField(window, ortbFieldAt(window, iw.idxBannerH, openrtbKeyH)))
	}
	if iw.idxVideoW >= 0 {
		hot.VideoW = uint16(parseJSONIntField(window, ortbFieldAt(window, iw.idxVideoW, openrtbKeyW)))
	}
	if iw.idxVideoH >= 0 {
		hot.VideoH = uint16(parseJSONIntField(window, ortbFieldAt(window, iw.idxVideoH, openrtbKeyH)))
	}
}

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
			hot.Flags |= OpenRTB26FlagSourceTID
		}
	}
}

func parsePMPPrivateFieldFromScan(impWin []byte, iw openrtb26ImpWinScan, hot *OpenRTB26Hot) {
	if len(impWin) == 0 || iw.idxPrivate < 0 {
		return
	}
	if parseJSONIntField(impWin, ortbFieldAt(impWin, iw.idxPrivate, openrtbKeyPrivate)) == 1 {
		hot.PMPPrivate = 1
		hot.Flags |= OpenRTB26FlagPMPPrivate
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
			hot.Flags |= OpenRTB26FlagSitePage
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
			hot.Flags |= OpenRTB26FlagAppVer
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
		hot.Flags |= OpenRTB26FlagEID
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
		hot.Flags |= OpenRTB26FlagMetric
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

func parseFirstImpIDAt(payload []byte, impIdx int, dst []byte) uint8 {
	if impIdx < 0 {
		return 0
	}
	i := impIdx + len(openrtbKeyImp)
	n := len(payload)
	if i >= n {
		return 0
	}
	_ = payload[n-1]
	for i < n && payload[i] != '[' {
		i++
	}
	if i >= n {
		return 0
	}
	i++
	for i < n && payload[i] != ']' {
		if bytes.HasPrefix(payload[i:], openrtbKeyID) {
			return uint8(parseQuotedField(payload, i+len(openrtbKeyID), dst))
		}
		i++
	}
	return 0
}

func parseImpObjectCountAt(payload []byte, impIdx int) int {
	if impIdx < 0 {
		return 0
	}
	i := impIdx + len(openrtbKeyImp)
	n := len(payload)
	if i >= n {
		return 0
	}
	_ = payload[n-1]
	for i < n && payload[i] != '[' {
		i++
	}
	if i >= n {
		return 0
	}
	i++
	count := 0
	depth := 0
	for i < n {
		switch payload[i] {
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

func sectionWindow(payload []byte, start int, maxLen int) []byte {
	if start < 0 || start >= len(payload) {
		return nil
	}
	end := start + maxLen
	if end > len(payload) {
		end = len(payload)
	}
	return payload[start:end]
}

func normalizeRegionBytes(src []byte, dst []byte) int {
	if len(src) == 0 || len(dst) == 0 {
		return 0
	}
	_ = src[len(src)-1]
	_ = dst[len(dst)-1]
	if len(src) >= 2 {
		if len(dst) < 2 {
			return 0
		}
		dst[0] = asciiUpperByte(src[0])
		dst[1] = asciiUpperByte(src[1])
		return 2
	}
	if len(dst) < 1 {
		return 0
	}
	dst[0] = asciiUpperByte(src[0])
	return 1
}

func normalizeCountryBytes(src []byte, dst []byte) int {
	if len(src) == 0 || len(dst) == 0 {
		return 0
	}
	_ = src[len(src)-1]
	_ = dst[len(dst)-1]
	if len(src) == 3 && asciiUpperByte(src[0]) == 'U' && asciiUpperByte(src[1]) == 'S' && asciiUpperByte(src[2]) == 'A' {
		if len(dst) < 2 {
			return 0
		}
		dst[0] = 'U'
		dst[1] = 'S'
		return 2
	}
	if len(src) >= 2 {
		if len(dst) < 2 {
			return 0
		}
		dst[0] = asciiUpperByte(src[0])
		dst[1] = asciiUpperByte(src[1])
		return 2
	}
	if len(dst) < 1 {
		return 0
	}
	dst[0] = asciiUpperByte(src[0])
	return 1
}

func OpenRTBDeviceType(dt int64) uint8 {
	switch dt {
	case 1, 4:
		return 2
	case 2:
		return 1
	case 5:
		return 4
	default:
		return 1
	}
}

func ParseQuotedField(payload []byte, start int, dst []byte) int {
	return parseQuotedField(payload, start, dst)
}

func parseQuotedField(payload []byte, start int, dst []byte) int {
	n := len(payload)
	if start >= n {
		return 0
	}
	i := start
	for i < n {
		c := payload[i]
		if c != ' ' && c != '\t' && c != ':' {
			break
		}
		i++
	}
	if i >= n || payload[i] != '"' {
		return 0
	}
	bud := newJSONScanBudget()
	fieldStart := i + 1
	end, ok := scanJSONStringEnd(payload, i, n, &bud)
	if !ok {
		return 0
	}
	ln := end - 1 - fieldStart
	if ln <= 0 {
		return 0
	}
	if dst == nil {
		return ln
	}
	if ln > len(dst) {
		return 0
	}
	copy(dst[:ln], payload[fieldStart:end-1])
	return ln
}

func DeadlineMonoFromTmax(tmaxMs int32) int64 {
	if tmaxMs <= 0 {
		tmaxMs = 200
	}
	return monoNano() + int64(tmaxMs)*int64(time.Millisecond)
}

func parseJSONIntField(payload []byte, start int) int64 {
	n := len(payload)
	if start >= n {
		return 0
	}
	var val int64
	digits := false
	for i := start; i < n; i++ {
		c := payload[i]
		if c >= '0' && c <= '9' {
			val = val*10 + int64(c-'0')
			digits = true
			continue
		}
		if digits {
			return val
		}
		if c != ' ' && c != '\t' && c != ':' {
			return 0
		}
	}
	return val
}

func parseDecimalMicroField(payload []byte, start int) int64 {
	n := len(payload)
	i := start
	for i < n && (payload[i] == ' ' || payload[i] == '\t' || payload[i] == ':') {
		i++
	}
	valStart := i
	for i < n && ((payload[i] >= '0' && payload[i] <= '9') || payload[i] == '.') {
		i++
	}
	if valStart < i {
		return parseDecimalMicro(payload[valStart:i])
	}
	return 0
}

func parseCategoryMaskFromArray(payload []byte, catIdx int) uint64 {
	n := len(payload)
	i := catIdx + len(openrtbKeyCat)
	for i < n && payload[i] != '[' {
		i++
	}
	if i >= n {
		return 1
	}
	i++
	var mask uint64
	for i < n && payload[i] != ']' {
		if payload[i] == '"' {
			i++
			start := i
			for i < n && payload[i] != '"' {
				i++
			}
			if start < i && i-start > 0 {
				d := payload[i-1]
				if d >= '0' && d <= '9' {
					mask |= uint64(1) << uint64(d-'0')
				} else {
					mask |= 1
				}
			}
			i++
			continue
		}
		i++
	}
	if mask == 0 {
		return 1
	}
	return mask
}
