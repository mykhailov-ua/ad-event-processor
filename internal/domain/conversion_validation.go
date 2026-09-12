package domain

import (
	"encoding/json"
	"strings"
)

const (
	ConversionValidationPendingKey        = "conversion_validation_pending"
	ConversionStatusSchemeSkipOutboundKey = "status_scheme_skip_outbound"
)

func ConversionValidationPending(payload []byte) bool {
	if len(payload) == 0 {
		return false
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return false
	}
	val, ok := raw[ConversionValidationPendingKey]
	if !ok || len(val) == 0 {
		return false
	}
	var b bool
	if err := json.Unmarshal(val, &b); err != nil {
		return false
	}
	return b
}

func ConversionSkipsOutboundPostback(payload []byte) bool {
	if len(payload) == 0 {
		return false
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return false
	}
	val, ok := raw[ConversionStatusSchemeSkipOutboundKey]
	if !ok || len(val) == 0 {
		return false
	}
	var s string
	if err := json.Unmarshal(val, &s); err == nil {
		v := strings.ToLower(strings.TrimSpace(s))
		return v == "true" || v == "1"
	}
	var b bool
	if err := json.Unmarshal(val, &b); err != nil {
		return false
	}
	return b
}
