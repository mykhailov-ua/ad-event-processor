package flow

import (
	"encoding/json"
	"strings"
)

func ParsePaths(raw json.RawMessage) ([]PathDTO, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var paths []PathDTO
	if err := json.Unmarshal(raw, &paths); err != nil {
		return nil, err
	}
	return paths, nil
}

func PathDBValidationErrors(err error) []PathErrorDTO {
	if err == nil {
		return nil
	}
	msg := err.Error()
	code := "flow_reference"
	switch {
	case strings.Contains(msg, "lander"):
		code = "dead_lander_ref"
	case strings.Contains(msg, "offer"):
		code = "dead_offer_ref"
	}
	return []PathErrorDTO{{
		PathIndex: -1,
		Code:      code,
		Message:   msg,
	}}
}
