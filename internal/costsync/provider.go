package costsync

import (
	"context"
	"time"
)

type Provider interface {
	Network() string
	Fetch(ctx context.Context, cred Credential, date time.Time) ([]CostLine, error)
}

type OAuthRefresher interface {
	Refresh(ctx context.Context, cred Credential) (accessToken string, expiresAt time.Time, err error)
}
