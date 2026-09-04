package campaign

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const CampaignExportBatchMaxIDs = 50

type CampaignExportBatchResponse struct {
	Items  map[string]CampaignExportBundle   `json:"items"`
	Errors []CampaignExportBatchResultRowDTO `json:"errors,omitempty"`
}

type CampaignExportBatchResultRowDTO struct {
	ID        string `json:"id"`
	ErrorCode string `json:"error_code,omitempty"`
}

func ParseCampaignExportIDs(raw string) ([]uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, invalidQueryError("ids required")
	}
	parts := strings.Split(raw, ",")
	if len(parts) > CampaignExportBatchMaxIDs {
		return nil, invalidQueryError(fmt.Sprintf("too many ids (max %d)", CampaignExportBatchMaxIDs))
	}
	ids := make([]uuid.UUID, 0, len(parts))
	seen := make(map[uuid.UUID]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := uuid.Parse(part)
		if err != nil {
			return nil, invalidQueryError("invalid campaign id")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, invalidQueryError("ids required")
	}
	return ids, nil
}

func ExportBatchErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrCampaignNotFound):
		return "not_found"
	case errors.Is(err, ErrForbidden):
		return "forbidden"
	default:
		return "export_failed"
	}
}
