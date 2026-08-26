package domain

import "encoding/json"

const ConversionValidationPendingKey = "conversion_validation_pending"

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
