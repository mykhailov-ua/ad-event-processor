package ingestion

import (
	"strings"

	"ad-event-processor/internal/domain"
)

const (
	flowDeviceDesktop uint8 = 1 << iota
	flowDeviceMobile
	flowDeviceTablet

	flowOSAndroid uint8 = 1 << iota
	flowOSIOS
	flowOSWindows
	flowOSMacOS
	flowOSLinux
)

type FlowPathFilters struct {
	Countries [][2]byte
	Devices   uint8
	OSMask    uint8
	Languages [][2]byte
}

type FlowSelectContext struct {
	Country    [2]byte
	DeviceMask uint8
	OSMask     uint8
	Language   [2]byte
}

type flowPathFiltersJSON struct {
	Countries []string `json:"countries"`
	Devices   []string `json:"devices"`
	OS        []string `json:"os"`
	Languages []string `json:"languages"`
}

func compileFlowPathFilters(raw *flowPathFiltersJSON) FlowPathFilters {
	if raw == nil {
		return FlowPathFilters{}
	}
	out := FlowPathFilters{}
	for _, code := range raw.Countries {
		code = strings.ToUpper(strings.TrimSpace(code))
		if len(code) != 2 {
			continue
		}
		out.Countries = append(out.Countries, [2]byte{code[0], code[1]})
	}
	for _, device := range raw.Devices {
		switch strings.ToLower(strings.TrimSpace(device)) {
		case "desktop":
			out.Devices |= flowDeviceDesktop
		case "mobile":
			out.Devices |= flowDeviceMobile
		case "tablet":
			out.Devices |= flowDeviceTablet
		}
	}
	for _, osName := range raw.OS {
		switch strings.ToLower(strings.TrimSpace(osName)) {
		case "android":
			out.OSMask |= flowOSAndroid
		case "ios":
			out.OSMask |= flowOSIOS
		case "windows":
			out.OSMask |= flowOSWindows
		case "macos":
			out.OSMask |= flowOSMacOS
		case "linux":
			out.OSMask |= flowOSLinux
		}
	}
	for _, lang := range raw.Languages {
		lang = strings.ToLower(strings.TrimSpace(lang))
		if len(lang) != 2 {
			continue
		}
		out.Languages = append(out.Languages, [2]byte{lang[0], lang[1]})
	}
	return out
}

func flowPathFiltersMatch(filters FlowPathFilters, ctx FlowSelectContext) bool {
	if len(filters.Countries) > 0 {
		if ctx.Country[0] == 0 || !flowCountryAllowed(ctx.Country, filters.Countries) {
			return false
		}
	}
	if filters.Devices != 0 {
		if ctx.DeviceMask == 0 || (filters.Devices&ctx.DeviceMask) == 0 {
			return false
		}
	}
	if filters.OSMask != 0 {
		if ctx.OSMask == 0 || (filters.OSMask&ctx.OSMask) == 0 {
			return false
		}
	}
	if len(filters.Languages) > 0 {
		if ctx.Language[0] == 0 || !flowLanguageAllowed(ctx.Language, filters.Languages) {
			return false
		}
	}
	return true
}

func flowCountryAllowed(country [2]byte, allowed [][2]byte) bool {
	for i := range allowed {
		if allowed[i] == country {
			return true
		}
	}
	return false
}

func flowLanguageAllowed(lang [2]byte, allowed [][2]byte) bool {
	for i := range allowed {
		if allowed[i] == lang {
			return true
		}
	}
	return false
}

func flowSelectContextFromEvent(evt *domain.Event) FlowSelectContext {
	if evt == nil {
		return FlowSelectContext{}
	}
	return FlowSelectContext{
		Country:    flowCountryBytes(evt.GeoCountry),
		DeviceMask: flowDeviceMaskFromUA(evt.UA),
		OSMask:     flowOSMaskFromUA(evt.UA),
		Language:   flowLanguageBytes(evt.AcceptLang),
	}
}

func flowCountryBytes(country string) [2]byte {
	country = strings.ToUpper(strings.TrimSpace(country))
	if len(country) != 2 {
		return [2]byte{}
	}
	return [2]byte{country[0], country[1]}
}

func flowLanguageBytes(acceptLang string) [2]byte {
	acceptLang = strings.TrimSpace(acceptLang)
	if acceptLang == "" {
		return [2]byte{}
	}
	if i := strings.IndexByte(acceptLang, ','); i >= 0 {
		acceptLang = acceptLang[:i]
	}
	if i := strings.IndexByte(acceptLang, '-'); i >= 0 {
		acceptLang = acceptLang[:i]
	}
	if i := strings.IndexByte(acceptLang, ';'); i >= 0 {
		acceptLang = acceptLang[:i]
	}
	acceptLang = strings.ToLower(strings.TrimSpace(acceptLang))
	if len(acceptLang) != 2 {
		return [2]byte{}
	}
	return [2]byte{acceptLang[0], acceptLang[1]}
}

func flowDeviceMaskFromUA(ua string) uint8 {
	uaLower := strings.ToLower(ua)
	if uaLower == "" {
		return 0
	}
	if strings.Contains(uaLower, "ipad") || strings.Contains(uaLower, "tablet") {
		return flowDeviceTablet
	}
	if strings.Contains(uaLower, "mobile") || strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "android") {
		return flowDeviceMobile
	}
	return flowDeviceDesktop
}

func flowOSMaskFromUA(ua string) uint8 {
	uaLower := strings.ToLower(ua)
	if uaLower == "" {
		return 0
	}
	switch {
	case strings.Contains(uaLower, "android"):
		return flowOSAndroid
	case strings.Contains(uaLower, "iphone"), strings.Contains(uaLower, "ipad"), strings.Contains(uaLower, "ios"):
		return flowOSIOS
	case strings.Contains(uaLower, "windows"):
		return flowOSWindows
	case strings.Contains(uaLower, "mac os"), strings.Contains(uaLower, "macintosh"):
		return flowOSMacOS
	case strings.Contains(uaLower, "linux"):
		return flowOSLinux
	default:
		return 0
	}
}
