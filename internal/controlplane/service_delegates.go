package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/shardadmin"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func (s *Service) GetCampaignRow(ctx context.Context, id uuid.UUID) (db.Campaign, error) {
	return db.New(s.pool).GetCampaign(ctx, domain.ToUUID(id))
}

func (s *Service) CreateCustomer(ctx context.Context, id uuid.UUID, name string, balance int64, currency string) error {
	if err := s.EnforceDeploymentLicenseCampaignCap(ctx); err != nil {
		return err
	}
	_, err := db.New(s.pool).CreateCustomer(ctx, db.CreateCustomerParams{
		ID:       domain.ToUUID(id),
		Name:     name,
		Balance:  balance,
		Currency: currency,
	})
	if err == nil {
		s.AuditLog(ctx, nil, uuid.Nil, "CREATE_CUSTOMER", "customer", &id, platformadmin.AuditCreateCustomerChange{
			Name:    name,
			Balance: balance,
		}, nil)
	}
	return err
}

func (s *Service) GenerateIdempotencyHash(customerID uuid.UUID, payload []byte) (string, error) {
	h := sha256.New()
	h.Write([]byte(customerID.String()))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (s *Service) GetPool() *pgxpool.Pool {
	return s.pool
}

func (s *Service) PostgresGate() *shardadmin.PostgresGate {
	if s == nil {
		return nil
	}
	return s.postgresGate
}

func (s *Service) SetPool(pool *pgxpool.Pool) {
	s.pool = pool
}

func (s *Service) SetPaymentPool(pool *pgxpool.Pool) {
	s.paymentPool = pool
}

func (s *Service) SetPayment(api domain.PaymentAPI) {
	if s != nil {
		s.payment = api
	}
}

func (s *Service) SetOpsAlerter(alerter *opsadmin.OpsAlerter) {
	s.alerter = alerter
}

func (s *Service) CancelCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	return campaign.CancelCampaign(ctx, s.pool, campaignID, reason)
}

func (s *Service) FinalizeCancelledCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	feePercent := 0.0
	if s.cfg != nil {
		feePercent = s.cfg.Management.CancellationFeePercent
	}
	return campaign.FinalizeCancelledCampaign(ctx, s.pool, s, feePercent, s.requirePgFencing, campaignID, reason)
}

func (s *Service) finalizeDrainingCampaign(ctx context.Context, q db.Querier, campaignID uuid.UUID, camp db.Campaign, reason string) error {
	feePercent := 0.0
	if s.cfg != nil {
		feePercent = s.cfg.Management.CancellationFeePercent
	}
	return campaign.FinalizeDrainingCampaign(ctx, q, s, feePercent, campaignID, camp, reason)
}

func (s *Service) ledgerStore() *billingadmin.LedgerStore {
	return billingadmin.NewLedgerStore(s)
}

func (s *Service) RequirePgFencing(ctx context.Context) error {
	return s.requirePgFencing(ctx)
}

func (s *Service) IsPgUniqueViolation(err error) bool {
	return isPgUniqueViolation(err)
}

var _ billingadmin.LedgerHost = (*Service)(nil)

func (s *Service) ErrRefundExceedsTopup() error     { return ErrRefundExceedsTopup }
func (s *Service) ErrChargebackExceedsTopup() error { return ErrChargebackExceedsTopup }
func (s *Service) ErrChargebackReversalExceedsWithdrawn() error {
	return ErrChargebackReversalExceedsWithdrawn
}

func (s *Service) GetLedgerEntry(ctx context.Context, paymentIntentID uuid.UUID) (found bool, entry db.BalanceLedger, refundTotal, chargebackTotal, reversalTotal int64, err error) {
	return s.ledgerStore().GetLedgerEntry(ctx, paymentIntentID)
}

func (s *Service) GetLedgerEntries(ctx context.Context, paymentIntentIDs []uuid.UUID) (map[uuid.UUID]domain.PaymentLedgerEntry, error) {
	return s.ledgerStore().GetLedgerEntries(ctx, paymentIntentIDs)
}

func (s *Service) UpdateOverdraft(ctx context.Context, id uuid.UUID, newOverdraft int64) error {
	return s.ledgerStore().UpdateOverdraft(ctx, id, newOverdraft)
}

func (s *Service) TopUpBalance(ctx context.Context, customerID uuid.UUID, amount int64, idempotencyKey string) error {
	return s.ledgerStore().TopUpBalance(ctx, customerID, amount, idempotencyKey)
}

func (s *Service) ApplyPaymentCredit(ctx context.Context, customerID uuid.UUID, amount int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRef string) (bool, int64, error) {
	return s.ledgerStore().ApplyPaymentCredit(ctx, customerID, amount, ledgerIdempotencyKey, paymentIntentID, provider, providerRef)
}

func (s *Service) ApplyPaymentRefund(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRefundID string) (bool, int64, error) {
	return s.ledgerStore().ApplyPaymentRefund(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerRefundID)
}

func (s *Service) ApplyPaymentChargeback(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	return s.ledgerStore().ApplyPaymentChargeback(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID)
}

func (s *Service) ApplyPaymentChargebackReversal(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	return s.ledgerStore().ApplyPaymentChargebackReversal(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID)
}

func (s *Service) RedisShards() []redis.UniversalClient {
	if s == nil {
		return nil
	}
	return s.redisShards
}

func (s *Service) campaignUpdateChannel() string {
	if s.cfg != nil && s.cfg.CampaignUpdateChannel != "" {
		return s.cfg.CampaignUpdateChannel
	}
	return "campaigns:update"
}

func (s *Service) publishCampaignUpdate(ctx context.Context, campaignID string) error {
	var pubErr error
	if len(s.redisShards) > 0 {
		pubErr = shardadmin.PublishCampaignControlToAllShards(ctx, s.redisShards, s.campaignUpdateChannel(), campaignID, time.Time{})
	} else {
		pubErr = fmt.Errorf("no redis pubsub client available")
	}

	if s.cfg != nil && s.cfg.CampaignUpdateBrokerFallback && s.cfg.Broker.URL != "" {
		topic := s.cfg.CampaignUpdateBrokerTopic
		if topic == "" {
			topic = domain.DefaultCampaignUpdateBrokerTopic
		}
		timeout := time.Duration(s.cfg.Broker.TimeoutMs) * time.Millisecond
		if err := domain.PublishCampaignUpdateBroker(
			ctx,
			s.cfg.Broker.URL,
			s.cfg.Broker.RedisURL,
			topic,
			timeout,
			campaignID,
		); err != nil {
			slog.Warn("campaign update broker fallback publish failed", "err", err, "campaign_id", campaignID)
			if pubErr != nil {
				return fmt.Errorf("redis pubsub: %w; broker: %w", pubErr, err)
			}
		} else if pubErr != nil {
			slog.Warn("campaign update redis pubsub failed; broker fallback ok", "err", pubErr, "campaign_id", campaignID)
			return nil
		}
	}

	if pubErr != nil {
		slog.Warn("campaign update publish failed", "campaign_id", campaignID, "err", pubErr)
	}
	return pubErr
}

func (s *Service) redisClientForCampaign(campaignID uuid.UUID) redis.UniversalClient {
	if len(s.redisShards) == 0 {
		return nil
	}
	if len(s.redisShards) == 1 {
		return s.redisShards[0]
	}
	idx := s.sharder.GetShard(campaignID)
	return s.redisShards[idx%len(s.redisShards)]
}

func (s *Service) SetTCPControlPublisher(tcp TCPControlPublisher) {
	s.tcpControl = tcp
}

func (s *Service) publishRoutingCutover(ctx context.Context, routingEpoch int64, slotVersion int32) {
	if ss, ok := s.sharder.(*domain.StaticSlotSharder); ok {
		prev := ss.Snapshot()
		ss.SwapSnapshot(slotVersion, &prev.Table, routingEpoch)
	}
	if s.tcpControl != nil {
		s.tcpControl.PublishSnapshot(ctx, routingEpoch, slotVersion)
	}
	if s.cfg != nil && s.cfg.Broker.URL != "" {
		timeout := time.Duration(s.cfg.Broker.TimeoutMs) * time.Millisecond
		if timeout <= 0 {
			timeout = 3 * time.Second
		}
		if err := domain.PublishSlotMapReload(
			ctx,
			s.cfg.Broker.URL,
			s.cfg.Broker.RedisURL,
			s.cfg.SlotMapReloadTopic,
			timeout,
			slotVersion,
			routingEpoch,
		); err != nil {
			slog.Warn("elastic routing broker publish failed", "err", err)
		}
	}
	metrics.ElasticRoutingCutoverTotal.Inc()
}

func (s *Service) ListAuditLogRows(ctx context.Context, limit, offset int32) ([]db.AdminAuditLog, int64, error) {
	q := db.New(s.pool)
	return coldpath.PaginatedQuery(
		func() (int64, error) { return q.CountAuditLogs(ctx) },
		func() ([]db.AdminAuditLog, error) {
			return q.ListAuditPaginated(ctx, db.ListAuditPaginatedParams{
				Limit:  limit,
				Offset: offset,
			})
		},
	)
}

func (s *Service) AuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any) {
	platformadmin.AuditLog(ctx, s, q, adminID, action, targetType, targetID, changes, metadata)
}

func (s *Service) RunAuditCleaner(ctx context.Context, retention platformadmin.Days) {
	platformadmin.RunAuditCleaner(ctx, s, retention)
}
