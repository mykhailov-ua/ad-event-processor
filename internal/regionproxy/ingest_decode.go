package regionproxy

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type batchJSON struct {
	RegionCode  uint8  `json:"region_code"`
	NodeID      string `json:"node_id"`
	SourceEpoch uint32 `json:"source_epoch"`
	Seq         uint64 `json:"seq"`
	FactorU     string `json:"factor_u"`
	Payload     []byte `json:"payload"`
	OpID        string `json:"op_id,omitempty"`
}

func DecodeBatchJSON(body []byte) (BatchInput, error) {
	var in batchJSON
	if err := json.Unmarshal(body, &in); err != nil {
		return BatchInput{}, fmt.Errorf("invalid json")
	}
	factorU, err := uuid.Parse(in.FactorU)
	if err != nil {
		return BatchInput{}, fmt.Errorf("invalid factor_u")
	}
	var opID uuid.UUID
	if in.OpID != "" {
		opID, err = uuid.Parse(in.OpID)
		if err != nil {
			return BatchInput{}, fmt.Errorf("invalid op_id")
		}
	}
	return BatchInput{
		RegionCode:  in.RegionCode,
		NodeID:      in.NodeID,
		SourceEpoch: in.SourceEpoch,
		Seq:         in.Seq,
		FactorU:     factorU,
		Payload:     in.Payload,
		OpID:        opID,
	}, nil
}
