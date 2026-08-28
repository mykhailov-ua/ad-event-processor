package campaign

import (
	"fmt"
	"strings"

	db "ad-event-processor/internal/domain/db"
)

func NormalizeConversionMappings(mappings []ConversionMappingDTO) ([]ConversionMappingDTO, error) {
	if len(mappings) == 0 {
		return []ConversionMappingDTO{}, nil
	}
	seen := make(map[string]struct{}, len(mappings))
	out := make([]ConversionMappingDTO, 0, len(mappings))
	for i := range mappings {
		row := mappings[i]
		status := strings.ToLower(strings.TrimSpace(row.InboundStatus))
		if status == "" {
			return nil, fmt.Errorf("inbound_status is required")
		}
		goal := strings.TrimSpace(row.GoalName)
		if goal == "" {
			goal = status
		}
		if row.PayoutMicro < 0 {
			return nil, fmt.Errorf("payout_micro must be non-negative")
		}
		if _, dup := seen[status]; dup {
			return nil, fmt.Errorf("duplicate inbound_status %q", status)
		}
		seen[status] = struct{}{}
		out = append(out, ConversionMappingDTO{
			InboundStatus: status,
			GoalName:      goal,
			PayoutMicro:   row.PayoutMicro,
		})
	}
	return out, nil
}

func ConversionMappingToDTO(row *db.CampaignConversionMapping) ConversionMappingDTO {
	if row == nil {
		return ConversionMappingDTO{}
	}
	return ConversionMappingDTO{
		InboundStatus: row.InboundStatus,
		GoalName:      row.GoalName,
		PayoutMicro:   row.PayoutMicro,
	}
}
