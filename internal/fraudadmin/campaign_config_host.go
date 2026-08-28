package fraudadmin

import (
	"context"

	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CampaignFraudAuditChange struct {
	FraudThresholdPass       uint8
	FraudThresholdSuspect    uint8
	FraudThresholdIVT        uint8
	FraudThresholdBlock      uint8
	SilentRejectEnabled      bool
	BehaviorFlags            int32
	CanvasRetestEnabled      bool
	CgnatIPPolicyEnabled     bool
	AcceptLangGeoEnabled     bool
	JSONSerializationEnabled bool
}

type CampaignConfigHost interface {
	ConfigPool() *pgxpool.Pool
	ConfigClickHouse() *database.ClickHouseQuery
	ConfigActorID(ctx context.Context) uuid.UUID
	ConfigAuditUpdate(ctx context.Context, q db.Querier, adminID uuid.UUID, campaignID uuid.UUID, changes CampaignFraudAuditChange)
	ConfigResolvePresetThresholds(ctx context.Context, name string) (pass, suspect, ivt, block uint8, err error)
	ConfigEnqueueUpdateCampaignFraud(ctx context.Context, q db.Querier, campaignID uuid.UUID) error
}
