package ingestion

//go:noinline
func parseDeviceSection(payload []byte, sec openrtb26Sections, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	if sec.device < 0 {
		return
	}
	window := sectionWindow(payload, sec.device, 1024)
	ds := scanDeviceWindow(window)
	if ds.idxIP >= 0 {
		if parseQuotedField(window, ortbFieldAt(window, ds.idxIP, openrtbKeyIP), nil) > 0 {
			hot.Flags |= openrtb26FlagDeviceIP
		}
	}
	if ds.idxIPv6 >= 0 {
		if parseQuotedField(window, ortbFieldAt(window, ds.idxIPv6, openrtbKeyIPv6), nil) > 0 {
			hot.Flags |= openrtb26FlagDeviceIPv6
		}
	}
	if ds.idxUA >= 0 {
		if parseQuotedField(window, ortbFieldAt(window, ds.idxUA, openrtbKeyUA), nil) > 0 {
			hot.Flags |= openrtb26FlagDeviceUA
		}
	}
	if ds.idxCountry >= 0 {
		var buf [8]byte
		ln := parseQuotedField(window, ortbFieldAt(window, ds.idxCountry, openrtbKeyCountry), buf[:])
		if ln > 0 {
			norm := normalizeCountryBytes(buf[:ln], hot.GeoCountry[:])
			hot.GeoCountryLen = uint8(norm)
			if norm > 0 {
				hot.Flags |= openrtb26FlagGeoCountry
			}
		}
	}
	if ds.idxOS >= 0 {
		cold.DeviceOSLen = uint8(parseQuotedField(window, ortbFieldAt(window, ds.idxOS, openrtbKeyOS), cold.DeviceOS[:]))
		if cold.DeviceOSLen > 0 {
			hot.Flags |= openrtb26FlagDeviceOS
		}
	}
	if ds.idxLanguage >= 0 {
		cold.DeviceLangLen = uint8(parseQuotedField(window, ortbFieldAt(window, ds.idxLanguage, openrtbKeyLanguage), cold.DeviceLang[:]))
		if cold.DeviceLangLen > 0 {
			hot.Flags |= openrtb26FlagDeviceLang
		}
	}
	if ds.idxRegion >= 0 {
		var buf [8]byte
		if ln := parseQuotedField(window, ortbFieldAt(window, ds.idxRegion, openrtbKeyRegion), buf[:]); ln > 0 {
			norm := normalizeRegionBytes(buf[:ln], cold.GeoRegion[:])
			cold.GeoRegionLen = uint8(norm)
			if norm > 0 {
				hot.Flags |= openrtb26FlagGeoRegion
			}
		}
	}
	if ds.idxIFA >= 0 {
		cold.DeviceIFALen = uint8(parseQuotedField(window, ortbFieldAt(window, ds.idxIFA, openrtbKeyIFA), cold.DeviceIFA[:]))
		if cold.DeviceIFALen > 0 {
			hot.Flags |= openrtb26FlagDeviceIFA
		}
	}
	if ds.idxLMT >= 0 {
		if parseJSONIntField(window, ortbFieldAt(window, ds.idxLMT, openrtbKeyLMT)) == 1 {
			hot.DeviceLMT = 1
			hot.Flags |= openrtb26FlagDeviceLMT
		}
	}
	if ds.idxConnectiontype >= 0 {
		ct := parseJSONIntField(window, ortbFieldAt(window, ds.idxConnectiontype, openrtbKeyConnectiontype))
		if ct >= 0 && ct <= 255 {
			hot.ConnectionType = uint8(ct)
			hot.Flags |= openrtb26FlagConnectionType
		}
	}
	if sec.user >= 0 {
		userWin := sectionWindow(payload, sec.user, 384)
		_, buIdx := scanUserWindow(userWin)
		if buIdx >= 0 {
			cold.BuyerUIDLen = uint8(parseQuotedField(userWin, ortbFieldAt(userWin, buIdx, openrtbKeyBuyeruid), cold.BuyerUID[:]))
			if cold.BuyerUIDLen > 0 {
				hot.Flags |= openrtb26FlagBuyerUID
			}
		}
	}
}

func parseRegsFlagsFromScan(payload []byte, scan openrtb26Scan, hot *OpenRTB26Hot) {
	if scan.idxGDPR >= 0 {
		if parseJSONIntField(payload, scan.idxGDPR+len(openrtbKeyGDPR)) == 1 {
			hot.Flags |= openrtb26FlagGDPR
		}
	}
	if scan.idxUSPrivacy >= 0 {
		var buf [16]byte
		if ln := parseQuotedField(payload, scan.idxUSPrivacy+len(openrtbKeyUSPrivacy), buf[:]); ln > 0 && buf[0] == 'Y' {
			hot.Flags |= openrtb26FlagUSPrivacyY
		}
	}
}

func parseCurrencyFlagsFromScan(payload []byte, scan openrtb26Scan, hot *OpenRTB26Hot) {
	if scan.idxCur >= 0 {
		var buf [8]byte
		if ln := parseQuotedField(payload, scan.idxCur+len(openrtbKeyCur), buf[:]); ln >= 3 {
			if buf[0] == 'E' && buf[1] == 'U' && buf[2] == 'R' {
				hot.Flags |= openrtb26FlagEUR
			}
		}
	}
	if hot.Flags&openrtb26FlagEUR == 0 && scan.idxBidfloorcur >= 0 {
		var buf [8]byte
		if ln := parseQuotedField(payload, scan.idxBidfloorcur+len(openrtbKeyBidfloorcur), buf[:]); ln >= 3 {
			if buf[0] == 'E' && buf[1] == 'U' && buf[2] == 'R' {
				hot.Flags |= openrtb26FlagEUR
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
				hot.Flags |= openrtb26FlagHasDomain
			}
		}
	}
	if sec.app >= 0 {
		window := sectionWindow(payload, sec.app, 640)
		as := scanAppWindow(window)
		if as.idxBundle >= 0 {
			cold.AppBundleLen = uint8(parseQuotedField(window, ortbFieldAt(window, as.idxBundle, openrtbKeyBundle), cold.AppBundle[:]))
			if cold.AppBundleLen > 0 {
				hot.Flags |= openrtb26FlagHasBundle
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
