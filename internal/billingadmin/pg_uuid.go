package billingadmin

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func pgUUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	id, err := uuid.FromBytes(u.Bytes[:])
	if err != nil {
		return ""
	}
	return id.String()
}
