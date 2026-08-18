package controlplane

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type CampaignCreateSpec = CreateCampaignInput

func toTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
