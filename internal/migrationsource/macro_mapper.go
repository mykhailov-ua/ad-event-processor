package migrationsource

import (
	"strings"
	"unicode"
)

// MacroMapper applies static token maps to click URL query params.
type MacroMapper struct {
	bySource map[string]MacroEntry
}

// NewMacroMapper builds a mapper from macro entries.
func NewMacroMapper(entries []MacroEntry) *MacroMapper {
	by := make(map[string]MacroEntry, len(entries))
	for _, e := range entries {
		by[strings.TrimSpace(e.Source)] = e
	}
	return &MacroMapper{bySource: by}
}

// ApplyQueryParams rewrites query keys from mapped token values and records unmapped macros.
func (m *MacroMapper) ApplyQueryParams(raw map[string]string) (map[string]string, []Warning, string) {
	if m == nil {
		return raw, nil, ""
	}
	out := make(map[string]string, len(raw))
	var warnings []Warning
	var ingressParam string
	for key, val := range raw {
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}
		entry, ok := m.bySource[val]
		if ok {
			target := strings.TrimSpace(entry.TargetKey)
			if target == "" {
				target = key
			}
			out[target] = val
			if entry.IngressCost {
				ingressParam = target
			}
			continue
		}
		out[key] = val
		if looksLikeMacroToken(val) {
			warnings = append(warnings, Warning{
				Slug:    "unmapped_macro",
				Message: "click URL value " + val + " has no entry in macro map",
			})
		}
	}
	return out, warnings, ingressParam
}

func looksLikeMacroToken(val string) bool {
	if strings.Contains(val, "{{") || strings.Contains(val, "}}") {
		return true
	}
	if strings.HasPrefix(val, "{") && strings.HasSuffix(val, "}") {
		return true
	}
	if strings.HasPrefix(val, "__") && strings.HasSuffix(val, "__") {
		return true
	}
	for _, r := range val {
		if r > unicode.MaxASCII {
			return false
		}
	}
	if strings.Contains(val, "{") || strings.Contains(val, "}") {
		return true
	}
	return false
}
