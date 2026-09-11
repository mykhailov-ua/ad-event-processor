package export

import (
	"encoding/json"
	"strconv"
	"strings"
)

func parseLayerDesyncMinCount(payload json.RawMessage) uint8 {
	if len(payload) == 0 {
		return 2
	}
	var raw struct {
		LayerDesyncCount json.RawMessage `json:"layer_desync_count"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil || len(raw.LayerDesyncCount) == 0 {
		return 2
	}
	var asInt int
	if err := json.Unmarshal(raw.LayerDesyncCount, &asInt); err == nil {
		return clampLayerDesyncMinCount(asInt)
	}
	asStr := strings.TrimSpace(string(raw.LayerDesyncCount))
	if asStr == "" {
		return 2
	}
	parsed, err := strconv.Atoi(asStr)
	if err != nil {
		return 2
	}
	return clampLayerDesyncMinCount(parsed)
}

func clampLayerDesyncMinCount(n int) uint8 {
	if n < 1 {
		return 1
	}
	if n > 255 {
		return 255
	}
	return uint8(n)
}
