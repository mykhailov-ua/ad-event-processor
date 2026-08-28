package controlplane

import (
	"time"

	"ad-event-processor/internal/campaign"

	"github.com/jackc/pgx/v5/pgtype"
)

type CampaignCreateSpec = campaign.CreateCampaignSpec
type CreateCampaignInput = campaign.CreateCampaignSpec

func toTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
