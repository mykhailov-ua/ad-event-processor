package domain

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func decoyLanderIDFromPg(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	return uuid.UUID(id.Bytes)
}

func applyCampaignDecoyFields(camp *Campaign, decoyLanderID pgtype.UUID) {
	if camp == nil {
		return
	}
	camp.DecoyLanderID = decoyLanderIDFromPg(decoyLanderID)
}

func ToPgDecoyLanderID(id uuid.UUID) pgtype.UUID {
	if id == uuid.Nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: id, Valid: true}
}
