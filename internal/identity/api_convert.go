package identity

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func uuidFromPg(u pgtype.UUID) uuid.UUID {
	return uuid.UUID(u.Bytes)
}
