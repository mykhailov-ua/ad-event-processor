package flow

import (
	"context"

	"ad-event-processor/pkg/landerhost"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HostedLanderHost interface {
	HostedLanderPool() *pgxpool.Pool
	HostedLanderStore() *landerhost.Store
	LanderPublicBase(ctx context.Context) string
	LanderPreviewSecret() []byte
	LanderManagementURL() string
	LanderMaxZipBytes() int64
}
