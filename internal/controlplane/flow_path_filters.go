package controlplane

import (
	"fmt"
	"strings"
)

type FlowPathFiltersDTO struct {
	Countries []string `json:"countries,omitempty"`
	Devices   []string `json:"devices,omitempty"`
	OS        []string `json:"os,omitempty"`
	Languages []string `json:"languages,omitempty"`
}

var allowedFlowPathDevices = map[string]struct{}{
	"desktop": {},
	"mobile":  {},
	"tablet":  {},
}

var allowedFlowPathOS = map[string]struct{}{
	"android": {},
	"ios":     {},
	"windows": {},
	"macos":   {},
	"linux":   {},
}

func validateFlowPathFilters(pathIndex int, filters *FlowPathFiltersDTO) error {
	if filters == nil {
		return nil
	}
	prefix := fmt.Sprintf("path %d filters", pathIndex+1)
	if len(filters.Countries) > 32 {
		return fmt.Errorf("%s: too many countries (max 32)", prefix)
	}
	seenCountry := make(map[string]struct{}, len(filters.Countries))
	for _, raw := range filters.Countries {
		code := strings.ToUpper(strings.TrimSpace(raw))
		if len(code) != 2 || code[0] < 'A' || code[0] > 'Z' || code[1] < 'A' || code[1] > 'Z' {
			return fmt.Errorf("%s: invalid country code %q", prefix, raw)
		}
		if _, ok := seenCountry[code]; ok {
			continue
		}
		seenCountry[code] = struct{}{}
	}
	if len(filters.Devices) > 3 {
		return fmt.Errorf("%s: too many devices (max 3)", prefix)
	}
	seenDevice := make(map[string]struct{}, len(filters.Devices))
	for _, raw := range filters.Devices {
		device := strings.ToLower(strings.TrimSpace(raw))
		if device == "" {
			return fmt.Errorf("%s: device is required", prefix)
		}
		if _, ok := allowedFlowPathDevices[device]; !ok {
			return fmt.Errorf("%s: invalid device %q (desktop, mobile, tablet)", prefix, raw)
		}
		seenDevice[device] = struct{}{}
	}
	if len(filters.OS) > 8 {
		return fmt.Errorf("%s: too many os values (max 8)", prefix)
	}
	seenOS := make(map[string]struct{}, len(filters.OS))
	for _, raw := range filters.OS {
		osName := strings.ToLower(strings.TrimSpace(raw))
		if osName == "" {
			return fmt.Errorf("%s: os is required", prefix)
		}
		if _, ok := allowedFlowPathOS[osName]; !ok {
			return fmt.Errorf("%s: invalid os %q (android, ios, windows, macos, linux)", prefix, raw)
		}
		seenOS[osName] = struct{}{}
	}
	if len(filters.Languages) > 16 {
		return fmt.Errorf("%s: too many languages (max 16)", prefix)
	}
	seenLang := make(map[string]struct{}, len(filters.Languages))
	for _, raw := range filters.Languages {
		lang := strings.ToLower(strings.TrimSpace(raw))
		if len(lang) != 2 || lang[0] < 'a' || lang[0] > 'z' || lang[1] < 'a' || lang[1] > 'z' {
			return fmt.Errorf("%s: invalid language code %q", prefix, raw)
		}
		if _, ok := seenLang[lang]; ok {
			continue
		}
		seenLang[lang] = struct{}{}
	}
	return nil
}
