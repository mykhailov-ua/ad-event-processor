package flow

import (
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
)

const (
	weightSumTarget    = 100.0
	weightSumTolerance = 0.01
	maxPaths           = 32
)

var allowedPathDevices = map[string]struct{}{
	"desktop": {},
	"mobile":  {},
	"tablet":  {},
}

var allowedPathOS = map[string]struct{}{
	"android": {},
	"ios":     {},
	"windows": {},
	"macos":   {},
	"linux":   {},
}

func ValidatePathShape(paths []PathDTO) error {
	if len(paths) == 0 {
		return fmt.Errorf("paths are required")
	}
	if len(paths) > maxPaths {
		return fmt.Errorf("too many paths (max %d)", maxPaths)
	}
	for i, path := range paths {
		if path.Weight <= 0 {
			return fmt.Errorf("path %d weight must be positive", i+1)
		}
		if len(path.Landers) == 0 {
			return fmt.Errorf("path %d requires at least one lander", i+1)
		}
		if len(path.Offers) == 0 {
			return fmt.Errorf("path %d requires at least one offer", i+1)
		}
		for j, lander := range path.Landers {
			if lander.LanderID == uuid.Nil {
				return fmt.Errorf("path %d lander %d id is required", i+1, j+1)
			}
			if lander.Weight <= 0 {
				return fmt.Errorf("path %d lander %d weight must be positive", i+1, j+1)
			}
		}
		for j, offer := range path.Offers {
			if offer.OfferID == uuid.Nil {
				return fmt.Errorf("path %d offer %d id is required", i+1, j+1)
			}
			if offer.Weight <= 0 {
				return fmt.Errorf("path %d offer %d weight must be positive", i+1, j+1)
			}
			if offer.CapDaily != nil && *offer.CapDaily <= 0 {
				return fmt.Errorf("path %d offer %d cap_daily must be positive", i+1, j+1)
			}
			if offer.CapTotal != nil && *offer.CapTotal <= 0 {
				return fmt.Errorf("path %d offer %d cap_total must be positive", i+1, j+1)
			}
		}
		if err := validatePathFilters(i, path.Filters); err != nil {
			return err
		}
	}
	return nil
}

func validatePathFilters(pathIndex int, filters *PathFiltersDTO) error {
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
		if _, ok := allowedPathDevices[device]; !ok {
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
		if _, ok := allowedPathOS[osName]; !ok {
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

func BuildValidateResponse(paths []PathDTO) ValidateResponseDTO {
	var errors []PathErrorDTO
	if err := ValidatePathShape(paths); err != nil {
		errors = append(errors, PathErrorDTO{
			PathIndex: -1,
			Code:      "invalid_shape",
			Message:   err.Error(),
		})
	}
	var weightSum float64
	for _, path := range paths {
		weightSum += float64(path.Weight)
	}
	if len(paths) > 0 && math.Abs(weightSum-weightSumTarget) > weightSumTolerance {
		errors = append(errors, PathErrorDTO{
			PathIndex: -1,
			Code:      "weight_sum",
			Message:   fmt.Sprintf("path weights must sum to 100, got %.2f", weightSum),
		})
	}
	resp := ValidateResponseDTO{Valid: len(errors) == 0, PathErrors: errors}
	if !resp.Valid && len(errors) == 1 && errors[0].Code == "weight_sum" {
		resp.SuggestedFixAction = "normalize_path_weights"
	}
	return resp
}

func FormatPathErrors(pathErrors []PathErrorDTO) string {
	if len(pathErrors) == 0 {
		return ""
	}
	parts := make([]string, 0, len(pathErrors))
	for _, pe := range pathErrors {
		if pe.Message != "" {
			parts = append(parts, pe.Message)
		}
	}
	return strings.Join(parts, "; ")
}
