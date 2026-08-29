package shard

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const CampaignEpochKey = "campaign_epoch"

func ToUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}
