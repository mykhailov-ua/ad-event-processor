package campaign

import (
	"context"
	"errors"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ImportExportHost interface {
	Pool() *pgxpool.Pool
	AssertMediaBuyerCampaignAccess(ctx context.Context, row db.Campaign) error
	GetFlow(ctx context.Context, flowID uuid.UUID) (FlowDTO, error)
	AuditImportCampaign(ctx context.Context, q *db.Queries, campaignID uuid.UUID, change ImportCampaignAuditChange, meta ImportCampaignIdempotencyMeta) error
	EmitCampaignLifecycleOutbox(ctx context.Context, q *db.Queries, campaignID uuid.UUID, status db.CampaignStatusType, budget int64) error
	PublishCampaignUpdate(ctx context.Context, campaignID string)
	PublishFlowReload(ctx context.Context)
}

type ImportCampaignAuditChange struct {
	Name string `json:"name"`
}

type ImportCampaignIdempotencyMeta struct {
	IdempotencyKey string `json:"idempotency_key"`
}

var (
	ErrCustomerNotFound      = errors.New("customer not found")
	ErrInsufficientBalance   = errors.New("insufficient balance")
	ErrIncompleteIdempotency = errors.New("incomplete idempotency")
)

func mapCampaignNotFound(err, notFound error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	return err
}

func flowIDOrNil(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
