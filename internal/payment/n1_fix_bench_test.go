package payment

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ad-event-processor/internal/database"
	paydb "ad-event-processor/internal/payment/db"
	"ad-event-processor/internal/payment/dbtest"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const n1DisputeListSize = 15

func setupPaymentDBWithQueryCounter(t testing.TB) (*pgxpool.Pool, *database.QueryCounter, func()) {
	t.Helper()
	ctx := context.Background()
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("payment_n1_db"),
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

	counter := &database.QueryCounter{}
	cfg, err := pgxpool.ParseConfig(connStr)
	require.NoError(t, err)
	cfg.ConnConfig.Tracer = counter

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)

	dbtest.ApplyMigrations(t, pool, dbtest.RepoRootJoin("internal/ingestion/migrations"))
	dbtest.ApplyMigrations(t, pool, dbtest.RepoRootJoin("internal/payment/migrations"))

	return pool, counter, func() {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
	}
}

func legacyListDisputesWithProvider(ctx context.Context, svc *Service) ([]DisputeListItem, error) {
	q := paydb.New(svc.pool)
	intents, err := q.ListDisputedPaymentIntents(ctx, paydb.ListDisputedPaymentIntentsParams{
		Limit:  n1DisputeListSize,
		Offset: 0,
	})
	if err != nil {
		return nil, err
	}
	items := make([]DisputeListItem, 0, len(intents))
	for i := range intents {
		intent := &intents[i]
		item := DisputeListItem{Intent: *intent}
		dispute, derr := q.GetLatestDisputeForIntent(ctx, intent.ID)
		if derr == nil {
			item.ProviderDisputeID = dispute.ProviderDisputeID
		}
		items = append(items, item)
	}
	return items, nil
}

func seedDisputedIntents(t testing.TB, pool *pgxpool.Pool, customerID uuid.UUID, n int) {
	t.Helper()
	ctx := context.Background()
	q := paydb.New(pool)
	for i := range n {
		intentID := uuid.New()
		_, err := q.CreatePaymentIntent(ctx, paydb.CreatePaymentIntentParams{
			ID:             pgtypeUUID(intentID),
			CustomerID:     pgtypeUUID(customerID),
			AmountMicro:    int64((i + 1) * 1_000_000),
			Currency:       "USD",
			Status:         paydb.PaymentPaymentIntentStatusDISPUTED,
			Provider:       "stripe",
			ProviderRef:    pgtypeText(fmt.Sprintf("pi_%d", i)),
			IdempotencyKey: fmt.Sprintf("idem-%d", i),
			Metadata:       []byte("{}"),
		})
		require.NoError(t, err)
		_, err = q.CreatePaymentDispute(ctx, paydb.CreatePaymentDisputeParams{
			ID:                pgtypeUUID(uuid.New()),
			PaymentIntentID:   pgtypeUUID(intentID),
			Provider:          "stripe",
			ProviderDisputeID: fmt.Sprintf("dp_%d", i),
			AmountMicro:       int64((i + 1) * 1_000_000),
			Status:            paydb.PaymentDisputeStatusOPEN,
		})
		require.NoError(t, err)
	}
}

func pgtypeUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func pgtypeText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

func TestN1Fix_ListDisputes_QueryCount(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	pool, counter, cleanup := setupPaymentDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	customerID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'disp-n1', 0, 'USD')`,
		pgtypeUUID(customerID))
	require.NoError(t, err)
	seedDisputedIntents(t, pool, customerID, n1DisputeListSize)

	svc := &Service{pool: pool}

	counter.Reset()
	_, err = legacyListDisputesWithProvider(ctx, svc)
	require.NoError(t, err)
	before := counter.Snapshot()

	counter.Reset()
	_, _, err = svc.ListDisputes(ctx, &customerID, n1DisputeListSize, 0)
	require.NoError(t, err)
	after := counter.Snapshot()

	t.Logf("8_list_disputes queries: before=%d after=%d (intents=%d)", before, after, n1DisputeListSize)
	require.Equal(t, int64(1+n1DisputeListSize), before)
	require.Equal(t, int64(3), after)
}
