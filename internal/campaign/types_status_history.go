package campaign

import (
	"time"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

type StatusHistoryDTO struct {
	ID         int64  `json:"id"`
	CampaignID string `json:"campaign_id"`
	OldStatus  string `json:"old_status,omitempty"`
	NewStatus  string `json:"new_status"`
	Reason     string `json:"reason,omitempty"`
	CreatedAt  string `json:"created_at"`
}

func statusHistoryToDTO(r db.CampaignStatusHistory) StatusHistoryDTO {
	var oldStatus string
	if r.OldStatus.Valid {
		oldStatus = string(r.OldStatus.CampaignStatusType)
	}
	return StatusHistoryDTO{
		ID:         r.ID,
		CampaignID: uuid.UUID(r.CampaignID.Bytes).String(),
		OldStatus:  oldStatus,
		NewStatus:  string(r.NewStatus),
		Reason:     r.Reason.String,
		CreatedAt:  r.CreatedAt.Time.Format(time.RFC3339),
	}
}
