package affiliatestatus

import (
	"bytes"
	"encoding/json"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxSchemaBodyBytes = 64 * 1024

type statusMappingDocument struct {
	StatusMap map[string]string `json:"status_map"`
}

func Map(statusMap map[string]string, external string) (string, bool) {
	if len(statusMap) == 0 {
		return "", false
	}
	mapped, ok := statusMap[strings.ToLower(strings.TrimSpace(external))]
	return mapped, ok
}

func MapFromSchemaBody(body []byte, external string) (string, bool) {
	doc, ok := ParseStatusMappingDocument(body)
	if !ok {
		return "", false
	}
	return Map(doc.StatusMap, external)
}

func ParseStatusMappingDocument(raw []byte) (statusMappingDocument, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || len(raw) > maxSchemaBodyBytes {
		return statusMappingDocument{}, false
	}
	jsonBytes := raw
	if raw[0] != '{' && raw[0] != '[' {
		var decoded any
		if err := yaml.Unmarshal(raw, &decoded); err != nil {
			return statusMappingDocument{}, false
		}
		marshaled, marshalErr := json.Marshal(decoded)
		if marshalErr != nil {
			return statusMappingDocument{}, false
		}
		jsonBytes = marshaled
	}
	var doc statusMappingDocument
	if err := json.Unmarshal(jsonBytes, &doc); err != nil {
		return statusMappingDocument{}, false
	}
	if len(doc.StatusMap) == 0 {
		return statusMappingDocument{}, false
	}
	return doc, true
}
