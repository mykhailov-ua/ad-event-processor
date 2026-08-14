package ingestion

import "bytes"

var (
	openrtbKeyBCat = []byte(`"bcat"`)
	openrtbKeyBAdv = []byte(`"badv"`)
	openrtbKeyBApp = []byte(`"bapp"`)
)

var rtbExchangeADomain = []byte("bidshard.local")

func parseBlocklistFieldsFromScan(payload []byte, scan openrtb26Scan, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	if scan.idxBCat >= 0 {
		cold.BCatCount = parseStringJSONArrayBCat(payload, scan.idxBCat+len(openrtbKeyBCat), cold)
		if cold.BCatCount > 0 {
			hot.Flags |= openrtb26FlagBCat
		}
	}
	if scan.idxBAdv >= 0 {
		cold.BAdvCount = parseStringJSONArrayBAdv(payload, scan.idxBAdv+len(openrtbKeyBAdv), cold)
		if cold.BAdvCount > 0 {
			hot.Flags |= openrtb26FlagBAdv
		}
	}
	if scan.idxBApp >= 0 {
		cold.BAppCount = parseStringJSONArrayBApp(payload, scan.idxBApp+len(openrtbKeyBApp), cold)
		if cold.BAppCount > 0 {
			hot.Flags |= openrtb26FlagBApp
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

func checkBlocklistsParsed(hot OpenRTB26Hot, cold *OpenRTB26Cold, blocklistEnforce bool) bool {
	if !blocklistEnforce || cold == nil {
		return false
	}
	if cold.BAppCount > 0 && cold.AppBundleLen > 0 {
		bundle := cold.AppBundle[:cold.AppBundleLen]
		for i := 0; i < int(cold.BAppCount); i++ {
			if cold.BAppLen[i] == 0 {
				continue
			}
			if bytes.EqualFold(cold.BApp[i][:cold.BAppLen[i]], bundle) {
				return true
			}
		}
	}
	if cold.BAdvCount > 0 {
		for i := 0; i < int(cold.BAdvCount); i++ {
			if cold.BAdvLen[i] == 0 {
				continue
			}
			if bytes.EqualFold(cold.BAdv[i][:cold.BAdvLen[i]], rtbExchangeADomain) {
				return true
			}
		}
	}
	_ = hot.Flags & openrtb26FlagBCat
	return false
}

func blockedCatMaskFromCold(cold *OpenRTB26Cold) uint64 {
	if cold == nil || cold.BCatCount == 0 {
		return 0
	}
	var mask uint64
	for i := 0; i < int(cold.BCatCount); i++ {
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
