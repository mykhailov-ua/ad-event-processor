package payment_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/controlplane"
	ads_db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/ingestion"
	"ad-event-processor/internal/payment"
	"ad-event-processor/internal/payment/db"
	"ad-event-processor/internal/payment/dbtest"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

const paymentContainerStopTimeout = 10 * time.Second

type FaultInfra struct {
	Pool            *pgxpool.Pool
	Redis           redis.UniversalClient
	PGContainer     *postgres.PostgresContainer
	RedisContainer  testcontainers.Container
	Cfg             *config.Config
	ControlplaneSvc *controlplane.Service
	SettlementGate  *SettlementFaultGate
}

type SeededPayment struct {
	CustomerID  uuid.UUID
	IntentID    uuid.UUID
	AmountMicro int64
	ProviderRef string
	OutboxID    int64
}

func SetupPaymentFaultInfra(t *testing.T) (*FaultInfra, func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("payment_fault_db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("secure_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(20*time.Second)),
	)
	require.NoError(t, err)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	dbtest.ApplyMigrations(t, pool, filepathJoinMigrations("internal/ingestion/migrations"))
	dbtest.ApplyMigrations(t, pool, filepathJoinMigrations("internal/payment/migrations"))

	redisContainer, err := rediscontainer.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)

	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	redisClient := redis.NewUniversalClient(&redis.UniversalOptions{Addrs: []string{endpoint}})
	require.NoError(t, redisClient.Ping(ctx).Err())

	cfg := &config.Config{
		PaymentInternalToken:    "payment_fault_token",
		SettlementInternalToken: "settlement_fault_token",
		StripeWebhookSecret:     "stripe_fault_wh_secret",
		MaxRetries:              3,
	}

	redisShards := []redis.UniversalClient{redisClient}
	mgmtSvc := controlplane.NewService(context.Background(), pool, redisShards, ingestion.NewStaticSlotSharder(len(redisShards)), cfg)
	settleHandler := controlplane.NewSettlementHandler(mgmtSvc, cfg)
	settlementGate := NewSettlementFaultGate(settleHandler.PaymentSettlement())

	infra := &FaultInfra{
		Pool:            pool,
		Redis:           redisClient,
		PGContainer:     pgContainer,
		RedisContainer:  redisContainer,
		Cfg:             cfg,
		ControlplaneSvc: mgmtSvc,
		SettlementGate:  settlementGate,
	}

	cleanup := func() {
		_ = redisClient.Close()
		pool.Close()
		_ = redisContainer.Terminate(ctx)
		_ = pgContainer.Terminate(ctx)
	}
	return infra, cleanup
}

func filepathJoinMigrations(rel string) string {
	return dbtest.RepoRootJoin(rel)
}

func StopPaymentContainer(t *testing.T, c testcontainers.Container) {
	t.Helper()
	timeout := paymentContainerStopTimeout
	require.NoError(t, c.Stop(context.Background(), &timeout))
}

func StartPaymentContainer(t *testing.T, c testcontainers.Container) {
	t.Helper()
	require.NoError(t, c.Start(context.Background()))
}

func (infra *FaultInfra) RefreshPGPool(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	infra.Pool.Close()
	connStr, err := infra.PGContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	infra.Pool = pool
	infra.ControlplaneSvc.SetPool(pool)
	require.Eventually(t, func() bool {
		return pool.Ping(ctx) == nil
	}, 30*time.Second, 200*time.Millisecond)
}

func (infra *FaultInfra) SetSettlementDown() {
	infra.SettlementGate.SetDown(true)
}

func (infra *FaultInfra) SetSettlementUp() {
	infra.SettlementGate.SetDown(false)
}

func NewOutboxWorkerForFault(infra *FaultInfra) *payment.OutboxWorker {
	worker := payment.NewOutboxWorker(infra.Pool, infra.Cfg)
	worker.SetSettlementAPI(infra.SettlementGate)
	return worker
}

func RequirePaymentFaultActive(t *testing.T, faultActive func() bool, msg string) {
	t.Helper()
	require.Eventually(t, faultActive, 10*time.Second, 100*time.Millisecond, msg)
}

func SeedCustomer(t *testing.T, pool *pgxpool.Pool, customerID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	_, err := ads_db.New(pool).CreateCustomer(ctx, ads_db.CreateCustomerParams{
		ID:       ingestion.ToUUID(customerID),
		Name:     "fault customer",
		Balance:  0,
		Currency: "USD",
	})
	require.NoError(t, err)
}

func SeedSucceededIntentWithOutbox(t *testing.T, infra *FaultInfra, customerID uuid.UUID, amountMicro int64, idempotencyKey string) SeededPayment {
	t.Helper()
	ctx := context.Background()
	SeedCustomer(t, infra.Pool, customerID)

	svc := payment.NewService(infra.Pool, infra.Cfg)
	result, err := svc.CreatePaymentIntent(ctx, customerID, amountMicro, "USD", idempotencyKey, nil)
	require.NoError(t, err)
	intent := result.Intent

	providerRef := intent.ProviderRef
	payload := fmt.Sprintf(`{"id":"evt_%s","type":"payment_intent.succeeded","data":{"object":{"id":%q,"amount":%d}}}`,
		idempotencyKey, providerRef, amountMicro)
	err = svc.ProcessStripeWebhook(ctx, "evt_"+idempotencyKey, "payment_intent.succeeded", []byte(payload), providerRef, amountMicro, payload)
	require.NoError(t, err)

	outboxRows, err := db.New(infra.Pool).GetPendingOutboxEventsForUpdate(ctx, 10)
	require.NoError(t, err)
	require.Len(t, outboxRows, 1)

	intentID, err := uuid.Parse(intent.ID)
	require.NoError(t, err)

	return SeededPayment{
		CustomerID:  customerID,
		IntentID:    intentID,
		AmountMicro: amountMicro,
		ProviderRef: providerRef,
		OutboxID:    outboxRows[0].ID,
	}
}

func SeedSettledIntent(t *testing.T, infra *FaultInfra, customerID uuid.UUID, amountMicro int64, idempotencyKey string) SeededPayment {
	t.Helper()
	seed := SeedSucceededIntentWithOutbox(t, infra, customerID, amountMicro, idempotencyKey)
	worker := NewOutboxWorkerForFault(infra)
	n, err := worker.ProcessOutbox(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, "PROCESSED", PaymentOutboxStatus(t, infra.Pool, seed.OutboxID))
	AssertPaymentFaultInvariants(t, infra.Pool, seed, seed.AmountMicro, 1)
	return seed
}

func ProcessRefundWebhook(t *testing.T, pool *pgxpool.Pool, svc *payment.Service, eventID, providerRef, refundID string, refundAmountMicro int64) int64 {
	t.Helper()
	stripeCents, err := payment.MicroToStripeAmount(refundAmountMicro)
	require.NoError(t, err)
	payload := fmt.Sprintf(`{"id":%q,"type":"refund.created","data":{"object":{"id":%q,"amount":%d,"payment_intent":%q,"status":"succeeded"}}}`,
		eventID, refundID, stripeCents, providerRef)
	err = svc.ProcessStripeRefundWebhook(context.Background(), eventID, "refund.created", []byte(payload), refundID, providerRef, refundAmountMicro, "succeeded")
	require.NoError(t, err)

	var outboxID int64
	require.NoError(t, pool.QueryRow(context.Background(), `
		SELECT id FROM payment.payment_outbox
		WHERE event_type = $1 AND status = 'PENDING'
		ORDER BY created_at DESC LIMIT 1`, payment.OutboxEventReverseBalance).Scan(&outboxID))
	return outboxID
}

func LedgerRefundCountForIntent(t *testing.T, pool *pgxpool.Pool, intentID uuid.UUID) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM balance_ledger
		WHERE payment_intent_id = $1 AND type = 'PAYMENT_REFUND'`, ingestion.ToUUID(intentID)).Scan(&n)
	require.NoError(t, err)
	return n
}

func AssertPaymentRefundInvariants(t *testing.T, pool *pgxpool.Pool, seed SeededPayment, wantBalance int64, wantRefundRows int) {
	t.Helper()
	require.Equal(t, wantBalance, CustomerBalance(t, pool, seed.CustomerID))
	require.Equal(t, wantRefundRows, LedgerRefundCountForIntent(t, pool, seed.IntentID))
}

func ProcessDisputeWebhook(t *testing.T, pool *pgxpool.Pool, svc *payment.Service, eventID, eventType, providerRef, disputeID string, amountMicro int64, stripeStatus string) {
	t.Helper()
	stripeCents, err := payment.MicroToStripeAmount(amountMicro)
	require.NoError(t, err)
	payload := fmt.Sprintf(`{"id":"%s","type":"%s","data":{"object":{"id":"%s","amount":%d,"payment_intent":"%s","status":"%s"}}}`,
		eventID, eventType, disputeID, stripeCents, providerRef, stripeStatus)
	err = svc.ProcessStripeDisputeWebhook(context.Background(), eventID, eventType, []byte(payload), disputeID, providerRef, amountMicro, stripeStatus)
	require.NoError(t, err)
}

func LatestOutboxIDByType(t *testing.T, pool *pgxpool.Pool, eventType string) int64 {
	t.Helper()
	var outboxID int64
	err := pool.QueryRow(context.Background(), `
		SELECT id FROM payment.payment_outbox
		WHERE event_type = $1 AND status = 'PENDING'
		ORDER BY created_at DESC LIMIT 1`, eventType).Scan(&outboxID)
	require.NoError(t, err)
	return outboxID
}

func LedgerChargebackCountForIntent(t *testing.T, pool *pgxpool.Pool, intentID uuid.UUID) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM balance_ledger
		WHERE payment_intent_id = $1 AND type = 'PAYMENT_CHARGEBACK'`, ingestion.ToUUID(intentID)).Scan(&n)
	require.NoError(t, err)
	return n
}

func LedgerChargebackReversalCountForIntent(t *testing.T, pool *pgxpool.Pool, intentID uuid.UUID) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM balance_ledger
		WHERE payment_intent_id = $1 AND type = 'PAYMENT_CHARGEBACK_REVERSAL'`, ingestion.ToUUID(intentID)).Scan(&n)
	require.NoError(t, err)
	return n
}

func AssertPaymentChargebackInvariants(t *testing.T, pool *pgxpool.Pool, seed SeededPayment, wantBalance int64, wantChargebackRows, wantReversalRows int) {
	t.Helper()
	require.Equal(t, wantBalance, CustomerBalance(t, pool, seed.CustomerID))
	require.Equal(t, wantChargebackRows, LedgerChargebackCountForIntent(t, pool, seed.IntentID))
	require.Equal(t, wantReversalRows, LedgerChargebackReversalCountForIntent(t, pool, seed.IntentID))
}

func CustomerBalance(t *testing.T, pool *pgxpool.Pool, customerID uuid.UUID) int64 {
	t.Helper()
	ctx := context.Background()
	cust, err := ads_db.New(pool).GetCustomerForUpdate(ctx, ingestion.ToUUID(customerID))
	require.NoError(t, err)
	return cust.Balance
}

func LedgerCountForIntent(t *testing.T, pool *pgxpool.Pool, intentID uuid.UUID) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM balance_ledger
		WHERE payment_intent_id = $1 AND type = 'PAYMENT_TOPUP'`, ingestion.ToUUID(intentID)).Scan(&n)
	require.NoError(t, err)
	return n
}

func PaymentOutboxStatus(t *testing.T, pool *pgxpool.Pool, outboxID int64) string {
	t.Helper()
	var status string
	err := pool.QueryRow(context.Background(), `
		SELECT status FROM payment.payment_outbox WHERE id = $1`, outboxID).Scan(&status)
	require.NoError(t, err)
	return status
}

func AssertPaymentFaultInvariants(t *testing.T, pool *pgxpool.Pool, seed SeededPayment, wantBalance int64, wantLedgerRows int) {
	t.Helper()
	require.Equal(t, wantBalance, CustomerBalance(t, pool, seed.CustomerID))
	require.Equal(t, wantLedgerRows, LedgerCountForIntent(t, pool, seed.IntentID))
}

func ItoaPaymentFault(n int) string {
	return strconv.Itoa(n)
}
