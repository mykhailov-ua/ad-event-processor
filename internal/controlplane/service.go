package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/money"
	"github.com/bidshard/ad-event-processor/pkg/pgfailover"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	pool            *pgxpool.Pool
	settlePoolField *pgxpool.Pool
	rdbs            []redis.UniversalClient
	sharder         domain.Sharder
	cfg             *config.Config
	pgGate          *PostgresGate
	alerter         *OpsAlerter
	chWrite         driver.Conn
	chQuery         *database.CHQuery
	paymentPool     *pgxpool.Pool
	payment         domain.PaymentAPI
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	workerMu        sync.Mutex
	closed          atomic.Bool
	locCache        sync.Map
	brokerDeltas    BrokerPendingDeltaReader
	tcpControl      *TCPControlServer
	nodeMetrics     *NodeMetricsWorker
	scoringWeights  *ScoringWeightsStore
	leaseWorker     *OperationLeaseWorker
	pgFencing       *pgfailover.FencingGate
	globalSpend     *GlobalSpendReconciler
	rtbBidShadeSim  RtbBidShadeSimulator
	shard0Mu        sync.Mutex
}

func (s *Service) SetRtbBidShadeSimulator(sim RtbBidShadeSimulator) {
	if s != nil {
		s.rtbBidShadeSim = sim
	}
}

func (s *Service) StartBackgroundWorker(fn func()) {
	s.startWorker(fn)
}

func (s *Service) CHQuery() *database.CHQuery {
	if s == nil {
		return nil
	}
	return s.chQuery
}

func (s *Service) CHWrite() driver.Conn {
	if s == nil {
		return nil
	}
	return s.chWrite
}

func (s *Service) startWorker(fn func()) {
	s.workerMu.Lock()
	if s.closed.Load() {
		s.workerMu.Unlock()
		return
	}
	s.wg.Add(1)
	s.workerMu.Unlock()

	go func() {
		defer s.wg.Done()
		fn()
	}()
}

func NewService(pool *pgxpool.Pool, rdbs []redis.UniversalClient, sharder domain.Sharder, cfg *config.Config) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		pool:    pool,
		rdbs:    rdbs,
		sharder: sharder,
		cfg:     cfg,
		ctx:     ctx,
		cancel:  cancel,
	}
	if cfg != nil {
		s.pgGate = NewPostgresGate(cfg.DBTrackerMaxConns)
	}
	s.startWorker(func() {
		if cfg == nil {
			return
		}
		if cfg.MultiRegionCell() {
			NewRegionOutboxRelay(s).Start(ctx, 20*time.Millisecond)
			return
		}
		if !cfg.MultiRegionGlobal() {
			NewOutboxWorker(s).Start(ctx, 20*time.Millisecond)
		}
	})
	s.startWorker(func() {
		adapter := s.dedupAdapter()
		if adapter == nil {
			return
		}
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := adapter.RejectStaleProposals(ctx); err != nil && ctx.Err() == nil {
					slog.Warn("dedup proposal janitor failed", "err", err)
				}
			}
		}
	})
	s.startWorker(func() {
		NewCampaignDrainWorker(s).Start(ctx, 20*time.Millisecond)
	})
	if cfg != nil && cfg.MultiRegionEnabled {
		worker := NewNodeMetricsWorker(s)
		s.nodeMetrics = worker
		s.startWorker(func() {
			worker.Start(ctx)
		})
		snapshotWorker := NewNodeMetricsSnapshotWorker(s)
		s.startWorker(func() {
			snapshotWorker.Start(ctx)
		})
		store, err := NewScoringWeightsStore(ctx, pool, cfg)
		if err != nil {
			slog.Error("scoring weights config invalid", "err", err)
			cancel()
			return nil
		}
		s.scoringWeights = store
		s.startWorker(func() {
			store.Start(ctx, pool, cfg)
		})
		leaseWorker := NewOperationLeaseWorker(s)
		s.leaseWorker = leaseWorker
		s.startWorker(func() {
			leaseWorker.Start(ctx)
		})
	}
	if cfg != nil && cfg.MultiRegionCell() {
		scorerWorker := NewNodeCapacityScorerWorker(s)
		s.startWorker(func() {
			scorerWorker.Start(ctx)
		})
	}
	if cfg != nil && cfg.MultiRegionGlobal() {
		globalScorerWorker := NewGlobalRegionTrafficScorerWorker(s)
		s.startWorker(func() {
			globalScorerWorker.Start(ctx)
		})
	}
	s.startWorker(func() {
		NewCreditScoringWorker(s).Start(ctx, 24*time.Hour)
	})
	s.startWorker(func() {
		NewScheduleWorker(s).Start(ctx)
	})
	s.startWorker(func() {
		NewSupplyAuditWorker(s).Start(ctx)
	})
	s.startWorker(func() {
		NewTLSImpersonationWorker(s).Start(ctx, 1*time.Hour)
	})
	s.startWorker(func() {
		NewSystemStateWorker(s).Start(ctx)
	})
	return s
}

func (s *Service) StartReconWorker(interval time.Duration) {
	s.startWorker(func() {
		NewReconWorker(s, interval).Start(s.ctx)
	})
}

func (s *Service) StartAuditCleaner(retention Days) {
	s.startWorker(func() {
		s.RunAuditCleaner(s.ctx, retention)
	})
}

func (s *Service) StartBlacklistJanitor(interval time.Duration) {
	s.startWorker(func() {
		NewBlacklistJanitor(s, interval).Start(s.ctx)
	})
}

func (s *Service) SetBrokerDeltas(reader BrokerPendingDeltaReader) {
	s.brokerDeltas = reader
}

func (s *Service) SetGlobalSpendReconciler(reconciler *GlobalSpendReconciler) {
	s.globalSpend = reconciler
}

func (s *Service) GlobalSpendReconciler() *GlobalSpendReconciler {
	if s == nil {
		return nil
	}
	return s.globalSpend
}

func (s *Service) RedisShards() []redis.UniversalClient {
	if s == nil {
		return nil
	}
	return s.rdbs
}

func (s *Service) GetPool() *pgxpool.Pool {
	return s.pool
}

func (s *Service) PgGate() *PostgresGate {
	if s == nil {
		return nil
	}
	return s.pgGate
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

func (s *Service) SetOpsAlerter(alerter *OpsAlerter) {
	s.alerter = alerter
}

func (s *Service) Close() {
	s.closed.Store(true)
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

func (s *Service) StartPacingController(syncWorkers []*domain.SyncWorker, interval time.Duration) {
	s.startWorker(func() {
		NewPacingControllerWorker(s, syncWorkers).Start(s.ctx, interval)
	})
}

func (s *Service) StartAutoscaleBudgetWorker(syncWorkers []*domain.SyncWorker, interval time.Duration) {
	s.startWorker(func() {
		NewAutoscaleBudgetWorker(s, syncWorkers).Start(s.ctx, interval)
	})
}

func (s *Service) StartDeliveryOptimizerWorker(syncWorkers []*domain.SyncWorker, interval time.Duration) {
	s.startWorker(func() {
		NewDeliveryOptimizerWorker(s, syncWorkers).Start(s.ctx, interval)
	})
}

func (s *Service) GetCampaignRow(ctx context.Context, id uuid.UUID) (db.Campaign, error) {
	return db.New(s.pool).GetCampaign(ctx, domain.ToUUID(id))
}

func (s *Service) CreateCustomer(ctx context.Context, id uuid.UUID, name string, balance int64, currency string) error {
	_, err := db.New(s.pool).CreateCustomer(ctx, db.CreateCustomerParams{
		ID:       domain.ToUUID(id),
		Name:     name,
		Balance:  balance,
		Currency: currency,
	})
	if err == nil {
		s.AuditLog(ctx, nil, uuid.Nil, "CREATE_CUSTOMER", "customer", &id, auditCreateCustomerChange{
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

func (s *Service) TopUpBalance(ctx context.Context, customerID uuid.UUID, amount int64, idempotencyKey string) error {
	if err := s.requirePgFencing(ctx); err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		_, err := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: idempotencyKey, Valid: true})
		if err == nil {
			return nil
		}
		_, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      domain.ToUUID(customerID),
			Balance: amount,
		})
		if err != nil {
			return fmt.Errorf("failed to update balance: %w", err)
		}
		_, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      domain.ToUUID(customerID),
			Amount:          amount,
			Type:            db.LedgerTypeTOPUP,
			IdempotencyHash: pgtype.Text{String: idempotencyKey, Valid: true},
			PaymentIntentID: pgtype.UUID{},
		})
		if err == nil {
			metrics.AddControlBalanceTopup("USD", money.APIValueFloat(amount))
			s.AuditLog(ctx, q, uuid.Nil, "TOPUP_BALANCE", "customer", &customerID, auditAmountChange{Amount: amount}, auditIdempotencyMeta{IdempotencyKey: idempotencyKey})
		}
		return err
	})
}

func (s *Service) ApplyPaymentCredit(ctx context.Context, customerID uuid.UUID, amount int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRef string) (bool, int64, error) {
	var ledgerEntryID int64
	var applied bool

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := db.New(tx)

		existingPI, err := q.GetLedgerByPaymentIntentForUpdate(ctx, domain.ToUUID(paymentIntentID))
		if err == nil {
			ledgerEntryID = existingPI.ID
			applied = false
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("payment intent idempotency check failed: %w", err)
		}

		existing, err := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: ledgerIdempotencyKey, Valid: true})
		if err == nil {
			ledgerEntryID = existing.ID
			applied = false
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("idempotency check failed: %w", err)
		}

		_, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      domain.ToUUID(customerID),
			Balance: amount,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrCustomerNotFound
			}
			return fmt.Errorf("failed to update balance: %w", err)
		}

		row, err := q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      domain.ToUUID(customerID),
			Amount:          amount,
			Type:            db.LedgerType("PAYMENT_TOPUP"),
			IdempotencyHash: pgtype.Text{String: ledgerIdempotencyKey, Valid: true},
			PaymentIntentID: domain.ToUUID(paymentIntentID),
		})
		if err != nil {
			if isPgUniqueViolation(err) {
				if existingPI, lookupErr := q.GetLedgerByPaymentIntentForUpdate(ctx, domain.ToUUID(paymentIntentID)); lookupErr == nil {
					ledgerEntryID = existingPI.ID
					applied = false
					return nil
				}
				if existing, lookupErr := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: ledgerIdempotencyKey, Valid: true}); lookupErr == nil {
					ledgerEntryID = existing.ID
					applied = false
					return nil
				}
			}
			return fmt.Errorf("failed to create ledger entry: %w", err)
		}

		ledgerEntryID = row.ID
		applied = true

		metrics.AddControlBalanceTopup("USD", money.APIValueFloat(amount))
		s.AuditLog(ctx, q, uuid.Nil, "PAYMENT_SETTLEMENT", "customer", &customerID, auditPaymentSettlementChange{
			Amount:          amount,
			PaymentIntentID: paymentIntentID.String(),
			Provider:        provider,
			ProviderRef:     providerRef,
		}, auditIdempotencyMeta{IdempotencyKey: ledgerIdempotencyKey})
		return nil
	})

	return applied, ledgerEntryID, err
}

func (s *Service) ApplyPaymentRefund(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRefundID string) (bool, int64, error) {
	if amountMicro <= 0 {
		return false, 0, errValidation("refund amount must be positive")
	}

	var ledgerEntryID int64
	var applied bool

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := db.New(tx)

		existing, err := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: ledgerIdempotencyKey, Valid: true})
		if err == nil {
			ledgerEntryID = existing.ID
			applied = false
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("refund idempotency check failed: %w", err)
		}

		topup, err := q.GetLedgerByPaymentIntentForUpdate(ctx, domain.ToUUID(paymentIntentID))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrPaymentTopupNotFound
			}
			return fmt.Errorf("payment topup lookup failed: %w", err)
		}

		refundedSoFar, err := q.SumPaymentRefundAmountForIntent(ctx, domain.ToUUID(paymentIntentID))
		if err != nil {
			return fmt.Errorf("sum payment refunds failed: %w", err)
		}
		if refundedSoFar+amountMicro > topup.Amount {
			return ErrRefundExceedsTopup
		}

		debitAmount := -amountMicro
		_, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      domain.ToUUID(customerID),
			Balance: debitAmount,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrCustomerNotFound
			}
			return fmt.Errorf("failed to debit balance: %w", err)
		}

		row, err := q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      domain.ToUUID(customerID),
			Amount:          debitAmount,
			Type:            db.LedgerType("PAYMENT_REFUND"),
			IdempotencyHash: pgtype.Text{String: ledgerIdempotencyKey, Valid: true},
			PaymentIntentID: domain.ToUUID(paymentIntentID),
		})
		if err != nil {
			if isPgUniqueViolation(err) {
				if existing, lookupErr := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: ledgerIdempotencyKey, Valid: true}); lookupErr == nil {
					ledgerEntryID = existing.ID
					applied = false
					return nil
				}
			}
			return fmt.Errorf("failed to create refund ledger entry: %w", err)
		}

		ledgerEntryID = row.ID
		applied = true

		s.AuditLog(ctx, q, uuid.Nil, "PAYMENT_REFUND", "customer", &customerID,
			auditPaymentRefundChange{
				Amount:           amountMicro,
				PaymentIntentID:  paymentIntentID.String(),
				Provider:         provider,
				ProviderRefundID: providerRefundID,
			},
			auditIdempotencyMeta{IdempotencyKey: ledgerIdempotencyKey})
		return nil
	})

	return applied, ledgerEntryID, err
}

func (s *Service) ApplyPaymentChargeback(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	return s.applyPaymentChargebackMovement(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID, "PAYMENT_CHARGEBACK", true)
}

func (s *Service) ApplyPaymentChargebackReversal(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	return s.applyPaymentChargebackMovement(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID, "PAYMENT_CHARGEBACK_REVERSAL", false)
}

func (s *Service) applyPaymentChargebackMovement(
	ctx context.Context,
	customerID uuid.UUID,
	amountMicro int64,
	ledgerIdempotencyKey string,
	paymentIntentID uuid.UUID,
	provider string,
	providerDisputeID string,
	ledgerType string,
	isDebit bool,
) (bool, int64, error) {
	if amountMicro <= 0 {
		return false, 0, errValidation("chargeback amount must be positive")
	}

	var ledgerEntryID int64
	var applied bool

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := db.New(tx)

		existing, err := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: ledgerIdempotencyKey, Valid: true})
		if err == nil {
			ledgerEntryID = existing.ID
			applied = false
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("chargeback idempotency check failed: %w", err)
		}

		topup, err := q.GetLedgerByPaymentIntentForUpdate(ctx, domain.ToUUID(paymentIntentID))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrPaymentTopupNotFound
			}
			return fmt.Errorf("payment topup lookup failed: %w", err)
		}

		refundedSoFar, err := q.SumPaymentRefundAmountForIntent(ctx, domain.ToUUID(paymentIntentID))
		if err != nil {
			return fmt.Errorf("sum payment refunds failed: %w", err)
		}
		chargebackSoFar, err := q.SumPaymentChargebackAmountForIntent(ctx, domain.ToUUID(paymentIntentID))
		if err != nil {
			return fmt.Errorf("sum payment chargebacks failed: %w", err)
		}
		reversalSoFar, err := q.SumPaymentChargebackReversalAmountForIntent(ctx, domain.ToUUID(paymentIntentID))
		if err != nil {
			return fmt.Errorf("sum payment chargeback reversals failed: %w", err)
		}

		netChargeback := chargebackSoFar - reversalSoFar
		if isDebit {
			if refundedSoFar+netChargeback+amountMicro > topup.Amount {
				return ErrChargebackExceedsTopup
			}
		} else if amountMicro > netChargeback {
			return ErrChargebackReversalExceedsWithdrawn
		}

		balanceDelta := amountMicro
		ledgerAmount := amountMicro
		if isDebit {
			balanceDelta = -amountMicro
			ledgerAmount = -amountMicro
		}

		_, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      domain.ToUUID(customerID),
			Balance: balanceDelta,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrCustomerNotFound
			}
			return fmt.Errorf("failed to update balance for chargeback: %w", err)
		}

		row, err := q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      domain.ToUUID(customerID),
			Amount:          ledgerAmount,
			Type:            db.LedgerType(ledgerType),
			IdempotencyHash: pgtype.Text{String: ledgerIdempotencyKey, Valid: true},
			PaymentIntentID: domain.ToUUID(paymentIntentID),
		})
		if err != nil {
			if isPgUniqueViolation(err) {
				if existing, lookupErr := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: ledgerIdempotencyKey, Valid: true}); lookupErr == nil {
					ledgerEntryID = existing.ID
					applied = false
					return nil
				}
			}
			return fmt.Errorf("failed to create chargeback ledger entry: %w", err)
		}

		ledgerEntryID = row.ID
		applied = true

		action := "PAYMENT_CHARGEBACK"
		if !isDebit {
			action = "PAYMENT_CHARGEBACK_REVERSAL"
		}
		s.AuditLog(ctx, q, uuid.Nil, action, "customer", &customerID,
			auditPaymentDisputeChange{
				Amount:            amountMicro,
				PaymentIntentID:   paymentIntentID.String(),
				Provider:          provider,
				ProviderDisputeID: providerDisputeID,
			},
			auditIdempotencyMeta{IdempotencyKey: ledgerIdempotencyKey})
		return nil
	})

	return applied, ledgerEntryID, err
}

func (s *Service) CancelCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		camp, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return err
		}
		if camp.Status == db.CampaignStatusTypeDELETED || camp.Status == db.CampaignStatusTypeDRAINING {
			return nil
		}
		_, err = q.UpdateCampaignStatus(ctx, db.UpdateCampaignStatusParams{
			ID:     domain.ToUUID(campaignID),
			Status: db.CampaignStatusTypeDRAINING,
		})
		if err != nil {
			return err
		}
		err = q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
			CampaignID: domain.ToUUID(campaignID),
			OldStatus:  db.NullCampaignStatusType{CampaignStatusType: camp.Status, Valid: true},
			NewStatus:  db.CampaignStatusTypeDRAINING,
			Reason:     pgtype.Text{String: reason, Valid: true},
		})
		if err == nil {
			payloadBytes, marshalErr := coldpath.MarshalOutbox(CampaignPayload{CampaignID: campaignID.String()})
			if marshalErr != nil {
				return fmt.Errorf("marshal cancel campaign outbox payload: %w", marshalErr)
			}
			_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "CANCEL_CAMPAIGN", Payload: payloadBytes})
		}
		return err
	})
}

func (s *Service) FinalizeCancelledCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		var camp db.Campaign
		err := tx.QueryRow(ctx, `
			SELECT status, budget_limit, current_spend, customer_id 
			FROM campaigns 
			WHERE id = $1 
			FOR UPDATE`, domain.ToUUID(campaignID)).Scan(&camp.Status, &camp.BudgetLimit, &camp.CurrentSpend, &camp.CustomerID)
		if err != nil {
			return err
		}
		return s.finalizeDrainingCampaign(ctx, q, campaignID, camp, reason)
	})
}

func (s *Service) finalizeDrainingCampaign(ctx context.Context, q db.Querier, campaignID uuid.UUID, camp db.Campaign, reason string) error {
	if camp.Status != db.CampaignStatusTypeDRAINING {
		return nil
	}
	totalBudget := camp.BudgetLimit
	currentSpend := camp.CurrentSpend
	remaining := totalBudget - currentSpend
	if remaining < 0 {
		remaining = 0
	}
	feePercent := 0.0
	if s.cfg != nil {
		feePercent = s.cfg.Management.CancellationFeePercent
	}
	fee := money.PercentFromFloat(remaining, feePercent)
	refund := remaining - fee
	if refund > 0 {
		_, err := q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      camp.CustomerID,
			Balance: refund,
		})
		if err != nil {
			return err
		}
		_, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      camp.CustomerID,
			CampaignID:      domain.ToUUID(campaignID),
			Amount:          refund,
			Type:            db.LedgerTypeRELEASE,
			PaymentIntentID: pgtype.UUID{},
		})
		if err != nil {
			return err
		}
	}
	if fee > 0 {
		_, err := q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      camp.CustomerID,
			CampaignID:      domain.ToUUID(campaignID),
			Amount:          fee,
			Type:            db.LedgerTypeFEE,
			PaymentIntentID: pgtype.UUID{},
		})
		if err != nil {
			return err
		}
		metrics.AddControlCommissionsCollected(money.APIValueFloat(fee))
	}
	if err := q.SoftDeleteCampaign(ctx, domain.ToUUID(campaignID)); err != nil {
		return err
	}
	if err := q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
		CampaignID: domain.ToUUID(campaignID),
		OldStatus:  db.NullCampaignStatusType{CampaignStatusType: db.CampaignStatusTypeDRAINING, Valid: true},
		NewStatus:  db.CampaignStatusTypeDELETED,
		Reason:     pgtype.Text{String: "Finalized", Valid: true},
	}); err != nil {
		return err
	}
	s.AuditLog(ctx, q, uuid.Nil, "CANCEL_CAMPAIGN", "campaign", &campaignID, auditReasonChange{Reason: reason}, nil)
	return nil
}

func (s *Service) campaignUpdateChannel() string {
	if s.cfg != nil && s.cfg.CampaignUpdateChannel != "" {
		return s.cfg.CampaignUpdateChannel
	}
	return "campaigns:update"
}

func (s *Service) publishCampaignUpdate(ctx context.Context, campaignID string) error {
	var pubErr error
	if len(s.rdbs) > 0 {
		pubErr = publishCampaignControlToAllShards(ctx, s.rdbs, s.campaignUpdateChannel(), campaignID, time.Time{})
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

func (s *Service) getRDB(campaignID uuid.UUID) redis.UniversalClient {
	if len(s.rdbs) == 0 {
		return nil
	}
	if len(s.rdbs) == 1 {
		return s.rdbs[0]
	}
	idx := s.sharder.GetShard(campaignID)
	return s.rdbs[idx%len(s.rdbs)]
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

func (s *Service) GetLedgerEntry(ctx context.Context, paymentIntentID uuid.UUID) (found bool, entry db.BalanceLedger, refundTotal, chargebackTotal, reversalTotal int64, err error) {
	q := db.New(s.pool)
	entry, err = q.GetLedgerByPaymentIntent(ctx, domain.ToUUID(paymentIntentID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			entry = db.BalanceLedger{}
		} else {
			return false, db.BalanceLedger{}, 0, 0, 0, err
		}
	} else {
		found = true
	}
	refundTotal, err = q.SumPaymentRefundAmountForIntent(ctx, domain.ToUUID(paymentIntentID))
	if err != nil {
		return false, db.BalanceLedger{}, 0, 0, 0, err
	}
	chargebackTotal, err = q.SumPaymentChargebackAmountForIntent(ctx, domain.ToUUID(paymentIntentID))
	if err != nil {
		return false, db.BalanceLedger{}, 0, 0, 0, err
	}
	reversalTotal, err = q.SumPaymentChargebackReversalAmountForIntent(ctx, domain.ToUUID(paymentIntentID))
	if err != nil {
		return false, db.BalanceLedger{}, 0, 0, 0, err
	}
	return found, entry, refundTotal, chargebackTotal, reversalTotal, nil
}

func (s *Service) UpdateOverdraft(ctx context.Context, id uuid.UUID, newOverdraft int64) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		cust, err := q.GetCustomerForUpdate(ctx, domain.ToUUID(id))
		if err != nil {
			return fmt.Errorf("failed to fetch customer for overdraft update: %w", err)
		}

		prevOverdraft := cust.AllowedOverdraft
		if newOverdraft == prevOverdraft {
			return nil
		}

		if newOverdraft < prevOverdraft {
			availableLimit := cust.Balance + newOverdraft
			if availableLimit < 0 {
				camps, err := q.ListCampaigns(ctx, db.ListCampaignsParams{
					Limit:      10000,
					Offset:     0,
					CustomerID: domain.ToUUID(id),
					Status:     pgtype.Text{String: string(db.CampaignStatusTypeACTIVE), Valid: true},
				})
				if err != nil {
					return fmt.Errorf("failed to list active campaigns for overdraft decrease: %w", err)
				}

				for _, c := range camps {
					if availableLimit >= 0 {
						break
					}

					locked, err := q.GetCampaignForUpdate(ctx, c.ID)
					if err != nil {
						return fmt.Errorf("failed to lock campaign for overdraft suspend: %w", err)
					}
					if locked.Status != db.CampaignStatusTypeACTIVE {
						continue
					}

					_, err = q.UpdateCampaignStatus(ctx, db.UpdateCampaignStatusParams{
						ID:     locked.ID,
						Status: db.CampaignStatusTypePAUSED,
					})
					if err != nil {
						return fmt.Errorf("failed to pause campaign: %w", err)
					}

					err = q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
						CampaignID: locked.ID,
						OldStatus:  db.NullCampaignStatusType{CampaignStatusType: db.CampaignStatusTypeACTIVE, Valid: true},
						NewStatus:  db.CampaignStatusTypePAUSED,
						Reason:     pgtype.Text{String: "Overdraft reduced, campaign suspended", Valid: true},
					})
					if err != nil {
						return fmt.Errorf("failed to write status history: %w", err)
					}

					budgetLimit := locked.BudgetLimit
					currentSpend := locked.CurrentSpend
					remaining := budgetLimit - currentSpend
					if remaining < 0 {
						remaining = 0
					}

					if remaining > 0 {
						_, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
							ID:      domain.ToUUID(id),
							Balance: remaining,
						})
						if err != nil {
							return fmt.Errorf("failed to refund balance for suspended campaign: %w", err)
						}

						_, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
							CustomerID:      domain.ToUUID(id),
							CampaignID:      locked.ID,
							Amount:          remaining,
							Type:            db.LedgerTypeRELEASE,
							PaymentIntentID: pgtype.UUID{},
						})
						if err != nil {
							return fmt.Errorf("failed to record release ledger entry: %w", err)
						}

						availableLimit += remaining
					}

					payloadBytes, marshalErr := coldpath.MarshalOutbox(CampaignPayload{CampaignID: uuid.UUID(locked.ID.Bytes).String()})
					if marshalErr != nil {
						return fmt.Errorf("marshal pause campaign outbox payload: %w", marshalErr)
					}
					_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
						EventType: "PAUSE_CAMPAIGN",
						Payload:   payloadBytes,
					})
					if err != nil {
						return fmt.Errorf("failed to emit outbox event for paused campaign: %w", err)
					}

					campID := uuid.UUID(locked.ID.Bytes)
					s.AuditLog(ctx, q, uuid.Nil, "SUSPEND_CAMPAIGN", "campaign", &campID, auditReasonChange{Reason: "overdraft_reduced"}, nil)
				}
			}
		}

		_, err = q.UpdateCustomerOverdraft(ctx, db.UpdateCustomerOverdraftParams{
			ID:               domain.ToUUID(id),
			AllowedOverdraft: newOverdraft,
		})
		if err != nil {
			return err
		}

		s.AuditLog(ctx, q, uuid.Nil, "UPDATE_CUSTOMER_OVERDRAFT", "customer", &id, auditOverdraftChange{
			OldOverdraft: money.FormatDecimal(prevOverdraft),
			NewOverdraft: money.FormatDecimal(newOverdraft),
		}, nil)
		return nil
	})
}

func (s *Service) SetTCPControlServer(tcp *TCPControlServer) {
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
