package identity

import (
	"espx/internal/identity/db"
)

func apiKeyFromListRow(row db.ListUserAPIKeysRow) APIKey {
	key := APIKey{
		ID:        uuidFromPg(row.ID).String(),
		Name:      row.Name,
		CreatedAt: row.CreatedAt.Time,
	}
	if row.ExpiresAt.Valid {
		t := row.ExpiresAt.Time
		key.ExpiresAt = &t
	}
	return key
}
