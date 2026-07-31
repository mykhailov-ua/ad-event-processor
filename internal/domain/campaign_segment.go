package domain

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func applyCampaignSegmentFields(
	camp *Campaign,
	retarget, include, exclude pgtype.UUID,
	ttlHours int32,
) {
	if camp == nil {
		return
	}
	if retarget.Valid {
		camp.RetargetSegmentID = uuid.UUID(retarget.Bytes)
	}
	camp.SegmentTTLHours = ttlHours
	if include.Valid {
		camp.SegmentIncludeID = uuid.UUID(include.Bytes)
	}
	if exclude.Valid {
		camp.SegmentExcludeID = uuid.UUID(exclude.Bytes)
	}
}
