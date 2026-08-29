package controlplane

import (
	"context"
	"strings"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/governance"
	"ad-event-processor/internal/outbox"
	"ad-event-processor/internal/reconciliation"
	"ad-event-processor/internal/shardadmin"
	"ad-event-processor/internal/supply"
	"ad-event-processor/internal/telegram"

	"github.com/google/uuid"
)

type (
	OutboxWorker             = outbox.Worker
	RegionOutboxRelay        = outbox.RegionOutboxRelay
	CampaignPayload          = outbox.CampaignPayload
	SettingsPayload          = outbox.SettingsPayload
	BlacklistPayload         = outbox.BlacklistPayload
	FraudThreatPayload       = outbox.FraudThreatPayload
	FraudModelVersionPayload = outbox.FraudModelVersionPayload
	PausePlacementPayload    = outbox.PausePlacementPayload
	CohortSnapshotPayload    = outbox.CohortSnapshotPayload
	RtbCatalogReloadPayload  = outbox.RtbCatalogReloadPayload

	brandIDPayload         = outbox.BrandIDPayload
	brandFcapOutboxPayload = outbox.BrandFcapPayload
	campaignPacingPayload  = outbox.CampaignPacingPayload
)

var (
	_ outbox.OutboxInfraHost = (*Service)(nil)
	_ outbox.OutboxRedisHost = (*Service)(nil)
	_ outbox.OutboxApplyHost = (*Service)(nil)
	_ outbox.Host            = (*Service)(nil)
	_ outbox.RegionRelayHost = (*regionRelayHost)(nil)
)

type regionRelayHost struct {
	*Service
}

func (h *regionRelayHost) RegionRelayRegionCode() uint8 {
	if h.Service == nil || h.Service.cfg == nil {
		return 0
	}
	return h.Service.cfg.RegionCode
}

func NewOutboxWorker(svc *Service) *OutboxWorker {
	return outbox.NewWorker(svc)
}

func NewRegionOutboxRelay(svc *Service) *RegionOutboxRelay {
	return outbox.NewRegionOutboxRelay(&regionRelayHost{Service: svc})
}

func (h *regionRelayHost) OperationLeaseWorker() *shardadmin.OperationLeaseWorker {
	if h.Service == nil {
		return nil
	}
	return h.Service.OperationLeaseWorker()
}

func (h *regionRelayHost) NewOperationLeaseWorker() *shardadmin.OperationLeaseWorker {
	if h.Service == nil {
		return nil
	}
	return NewOperationLeaseWorker(h.Service)
}

func (s *Service) MultiRegionCell() bool {
	return s != nil && s.cfg != nil && s.cfg.MultiRegionCell()
}

func (s *Service) SetNXOnAllShards(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return shardadmin.SetNXOnAllShards(ctx, s.redisShards, key, value, ttl)
}

func (s *Service) ExportSupplyFiles(ctx context.Context) error {
	return supply.ExportFiles(ctx, s)
}

func (s *Service) RelayDeliveryBookRequest(ctx context.Context, regionCode uint8, outboxEventID int64, eventType string, payload []byte, attempt int32) shardadmin.OperationLeaseBookRequest {
	return shardadmin.RelayDeliveryBookRequest(ctx, s, regionCode, outboxEventID, eventType, payload, attempt)
}

func (s *Service) NewOperationLeaseWorker() *shardadmin.OperationLeaseWorker {
	return NewOperationLeaseWorker(s)
}

func (s *Service) OutboxAlerter() outbox.Alerter {
	return s.alerter
}

func (s *Service) AuditActorID(ctx context.Context) uuid.UUID {
	if u, ok := GetUser(ctx); ok {
		return u.UserID
	}
	return uuid.Nil
}

func (s *Service) CampaignUpdateChannel() string {
	return s.campaignUpdateChannel()
}

func (s *Service) PublishRegistryFullSync(ctx context.Context) error {
	return s.publishCampaignUpdate(ctx, domain.RegistryFullSyncPayload)
}

func (s *Service) FlowReloadChannel() string {
	if s.cfg != nil && strings.TrimSpace(s.cfg.FlowReloadChannel) != "" {
		return strings.TrimSpace(s.cfg.FlowReloadChannel)
	}
	return "flow:reload"
}

func (s *Service) ApplyQuotaRepair(ctx context.Context, eventID int64, payload []byte) error {
	return governance.NewOutboxWorker(s).ApplyQuotaRepair(ctx, eventID, payload)
}

func (s *Service) ApplyReconciliationAdjust(ctx context.Context, eventID int64, payload []byte) error {
	return reconciliation.NewAdjustApplier(s).Apply(ctx, eventID, payload)
}

func (s *Service) HandleTelegramEvent(ctx context.Context, payload []byte) error {
	return telegram.NewService(s).HandleOutboxEvent(ctx, payload)
}
