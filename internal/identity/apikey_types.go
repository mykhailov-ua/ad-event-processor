package identity

import "time"

type APIKey struct {
	ID        string
	Name      string
	CreatedAt time.Time
	ExpiresAt *time.Time
}
