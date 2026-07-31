package ingestion

import (
	"espx/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const CampaignEpochKey = domain.CampaignEpochKey

const RtbFloorRedisKeyPrefix = domain.RtbFloorRedisKeyPrefix

func ToUUID(u uuid.UUID) pgtype.UUID {
	return domain.ToUUID(u)
}
