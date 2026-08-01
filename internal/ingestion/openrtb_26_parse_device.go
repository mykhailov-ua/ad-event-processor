package ingestion

import "bytes"

//go:noinline
func parseDeviceSection(payload []byte, sec openrtb26Sections, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	if sec.device < 0 {
		return
	}
	window := sectionWindow(payload, sec.device, 1024)
	if ipRel := bytes.Index(window, openrtbKeyIP); ipRel >= 0 {
		if parseQuotedField(window, ipRel+len(openrtbKeyIP), nil) > 0 {
			hot.Flags |= openrtb26FlagDeviceIP
		}
	}
	if v6Rel := bytes.Index(window, openrtbKeyIPv6); v6Rel >= 0 {
		if parseQuotedField(window, v6Rel+len(openrtbKeyIPv6), nil) > 0 {
			hot.Flags |= openrtb26FlagDeviceIPv6
		}
	}
	if uaRel := bytes.Index(window, openrtbKeyUA); uaRel >= 0 {
		if parseQuotedField(window, uaRel+len(openrtbKeyUA), nil) > 0 {
			hot.Flags |= openrtb26FlagDeviceUA
		}
	}
	if idx := bytes.Index(window, openrtbKeyCountry); idx >= 0 {
		var buf [8]byte
		ln := parseQuotedField(window, idx+len(openrtbKeyCountry), buf[:])
		if ln > 0 {
			norm := normalizeCountryBytes(buf[:ln], hot.GeoCountry[:])
			hot.GeoCountryLen = uint8(norm)
			if norm > 0 {
				hot.Flags |= openrtb26FlagGeoCountry
			}
		}
	}
	if osIdx := bytes.Index(window, openrtbKeyOS); osIdx >= 0 {
		cold.DeviceOSLen = uint8(parseQuotedField(window, osIdx+len(openrtbKeyOS), cold.DeviceOS[:]))
		if cold.DeviceOSLen > 0 {
			hot.Flags |= openrtb26FlagDeviceOS
		}
	}
	if langIdx := bytes.Index(window, openrtbKeyLanguage); langIdx >= 0 {
		cold.DeviceLangLen = uint8(parseQuotedField(window, langIdx+len(openrtbKeyLanguage), cold.DeviceLang[:]))
		if cold.DeviceLangLen > 0 {
			hot.Flags |= openrtb26FlagDeviceLang
		}
	}
	if regIdx := bytes.Index(window, openrtbKeyRegion); regIdx >= 0 {
		var buf [8]byte
		if ln := parseQuotedField(window, regIdx+len(openrtbKeyRegion), buf[:]); ln > 0 {
			norm := normalizeRegionBytes(buf[:ln], cold.GeoRegion[:])
			cold.GeoRegionLen = uint8(norm)
			if norm > 0 {
				hot.Flags |= openrtb26FlagGeoRegion
			}
		}
	}
	if ifaIdx := bytes.Index(window, openrtbKeyIFA); ifaIdx >= 0 {
		cold.DeviceIFALen = uint8(parseQuotedField(window, ifaIdx+len(openrtbKeyIFA), cold.DeviceIFA[:]))
		if cold.DeviceIFALen > 0 {
			hot.Flags |= openrtb26FlagDeviceIFA
		}
	}
	if lmtIdx := bytes.Index(window, openrtbKeyLMT); lmtIdx >= 0 {
		if parseJSONIntField(window, lmtIdx+len(openrtbKeyLMT)) == 1 {
			hot.DeviceLMT = 1
			hot.Flags |= openrtb26FlagDeviceLMT
		}
	}
	if ctIdx := bytes.Index(window, openrtbKeyConnectiontype); ctIdx >= 0 {
		ct := parseJSONIntField(window, ctIdx+len(openrtbKeyConnectiontype))
		if ct >= 0 && ct <= 255 {
			hot.ConnectionType = uint8(ct)
			hot.Flags |= openrtb26FlagConnectionType
		}
	}
	if sec.user >= 0 {
		userWin := sectionWindow(payload, sec.user, 384)
		if buIdx := bytes.Index(userWin, openrtbKeyBuyeruid); buIdx >= 0 {
			cold.BuyerUIDLen = uint8(parseQuotedField(userWin, buIdx+len(openrtbKeyBuyeruid), cold.BuyerUID[:]))
			if cold.BuyerUIDLen > 0 {
				hot.Flags |= openrtb26FlagBuyerUID
			}
		}
	}
}

func parseRegsFlags(payload []byte, hot *OpenRTB26Hot) {
	if idx := bytes.Index(payload, openrtbKeyGDPR); idx >= 0 {
		if parseJSONIntField(payload, idx+len(openrtbKeyGDPR)) == 1 {
			hot.Flags |= openrtb26FlagGDPR
		}
	}
	if idx := bytes.Index(payload, openrtbKeyUSPrivacy); idx >= 0 {
		var buf [16]byte
		if ln := parseQuotedField(payload, idx+len(openrtbKeyUSPrivacy), buf[:]); ln > 0 && buf[0] == 'Y' {
			hot.Flags |= openrtb26FlagUSPrivacyY
		}
	}
}

func parseCurrencyFlags(payload []byte, hot *OpenRTB26Hot) {
	if idx := bytes.Index(payload, openrtbKeyCur); idx >= 0 {
		var buf [8]byte
		if ln := parseQuotedField(payload, idx+len(openrtbKeyCur), buf[:]); ln >= 3 {
			if buf[0] == 'E' && buf[1] == 'U' && buf[2] == 'R' {
				hot.Flags |= openrtb26FlagEUR
			}
		}
	}
	if hot.Flags&openrtb26FlagEUR == 0 {
		if idx := bytes.Index(payload, openrtbKeyBidfloorcur); idx >= 0 {
			var buf [8]byte
			if ln := parseQuotedField(payload, idx+len(openrtbKeyBidfloorcur), buf[:]); ln >= 3 {
				if buf[0] == 'E' && buf[1] == 'U' && buf[2] == 'R' {
					hot.Flags |= openrtb26FlagEUR
				}
			}
		}
	}
}

func parseUserIDAt(payload []byte, userIdx int, dst []byte) uint8 {
	if userIdx < 0 {
		return 0
	}
	slice := sectionWindow(payload, userIdx, 384)
	if idRel := bytes.Index(slice, openrtbKeyID); idRel >= 0 {
		return uint8(parseQuotedField(slice, idRel+len(openrtbKeyID), dst))
	}
	return 0
}

func parseInventoryFieldsAt(payload []byte, sec openrtb26Sections, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	if sec.site >= 0 {
		window := sectionWindow(payload, sec.site, 640)
		if domIdx := bytes.Index(window, openrtbKeyDomain); domIdx >= 0 {
			cold.SiteDomainLen = uint8(parseQuotedField(window, domIdx+len(openrtbKeyDomain), cold.SiteDomain[:]))
			if cold.SiteDomainLen > 0 {
				hot.Flags |= openrtb26FlagHasDomain
			}
		}
	}
	if sec.app >= 0 {
		window := sectionWindow(payload, sec.app, 640)
		if bunIdx := bytes.Index(window, openrtbKeyBundle); bunIdx >= 0 {
			cold.AppBundleLen = uint8(parseQuotedField(window, bunIdx+len(openrtbKeyBundle), cold.AppBundle[:]))
			if cold.AppBundleLen > 0 {
				hot.Flags |= openrtb26FlagHasBundle
			}
		}
	}
}

func parseImpDimensionsAt(impIdx int, window []byte, hot *OpenRTB26Hot) {
	if impIdx < 0 || len(window) == 0 {
		return
	}
	if bIdx := bytes.Index(window, openrtbKeyBanner); bIdx >= 0 {
		bannerWin := sectionWindow(window, bIdx, 160)
		if wIdx := bytes.Index(bannerWin, openrtbKeyW); wIdx >= 0 {
			hot.BannerW = uint16(parseJSONIntField(bannerWin, wIdx+len(openrtbKeyW)))
		}
		if hIdx := bytes.Index(bannerWin, openrtbKeyH); hIdx >= 0 {
			hot.BannerH = uint16(parseJSONIntField(bannerWin, hIdx+len(openrtbKeyH)))
		}
	}
	if vIdx := bytes.Index(window, openrtbKeyVideo); vIdx >= 0 {
		videoWin := sectionWindow(window, vIdx, 200)
		if wIdx := bytes.Index(videoWin, openrtbKeyW); wIdx >= 0 {
			hot.VideoW = uint16(parseJSONIntField(videoWin, wIdx+len(openrtbKeyW)))
		}
		if hIdx := bytes.Index(videoWin, openrtbKeyH); hIdx >= 0 {
			hot.VideoH = uint16(parseJSONIntField(videoWin, hIdx+len(openrtbKeyH)))
		}
	}
}
