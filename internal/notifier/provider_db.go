package notifier

import (
	"espx/internal/notifier/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func MapDBProviderStringsToDB(providers []string) []db.NotifierProvider {
	if len(providers) == 0 {
		return nil
	}
	out := make([]db.NotifierProvider, 0, len(providers))
	for _, provider := range providers {
		out = append(out, db.NotifierProvider(provider))
	}
	return out
}

func pgUUIDFromString(id string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return pgtype.UUID{}, ErrInvalidNotificationID
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}
