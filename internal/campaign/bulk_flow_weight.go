package campaign

import (
	"encoding/json"
	"fmt"

	"ad-event-processor/internal/flow"
)

type BulkFlowPathWeight struct {
	PathIndex int32 `json:"path_index"`
	Weight    int32 `json:"weight"`
}

func ApplyFlowPathWeights(paths []FlowPathDTO, weights []BulkFlowPathWeight) ([]FlowPathDTO, error) {
	if len(weights) == 0 {
		return paths, nil
	}
	out := append([]FlowPathDTO(nil), paths...)
	for i, row := range weights {
		idx := int(row.PathIndex)
		if idx < 0 || idx >= len(out) {
			return nil, fmt.Errorf("flow_path_weights[%d]: path_index out of range", i)
		}
		if row.Weight <= 0 {
			return nil, fmt.Errorf("flow_path_weights[%d]: weight must be positive", i)
		}
		out[idx].Weight = row.Weight
	}
	if err := flow.ValidatePaths(out); err != nil {
		return nil, err
	}
	return out, nil
}

func MarshalFlowPaths(paths []FlowPathDTO) (json.RawMessage, error) {
	if len(paths) == 0 {
		return json.RawMessage("[]"), nil
	}
	raw, err := json.Marshal(paths)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}
