package controlplane

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"ad-event-processor/internal/brand"
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/campaign/runtime"
	"ad-event-processor/internal/campaign/wizard"
	campaignworker "ad-event-processor/internal/campaign/worker"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/fraud"
	"ad-event-processor/internal/marginguard"
	"ad-event-processor/internal/nodeadmin"
	"ad-event-processor/internal/notify"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/pgfailover"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/privacyadmin"
	"ad-event-processor/internal/reconciliation"
	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/rtbadmin"
	"ad-event-processor/internal/settingsadmin"
	"ad-event-processor/internal/shardadmin"
	"ad-event-processor/internal/supply"
	"ad-event-processor/pkg/domainhealth"
	"ad-event-processor/pkg/landerhost"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Service: control plane composition root; cold path only (PG, Redis shards, CH query, outbox).
type Service struct {
	pool                     *pgxpool.Pool
	settlementPostgresPool   *pgxpool.Pool
	redisShards              []redis.UniversalClient
	sharder                  domain.Sharder
	cfg                      *config.Config
	postgresGate             *shardadmin.PostgresGate
	alerter                  *opsadmin.OpsAlerter
	clickhouseWriteConn      driver.Conn
	clickhouseQuery          *database.ClickHouseQuery
	paymentPool              *pgxpool.Pool
	payment                  domain.PaymentAPI
	ctx                      context.Context
	cancel                   context.CancelFunc
	wg                       sync.WaitGroup
	workerMutex              sync.Mutex
	closed                   atomic.Bool
	timezoneLocationCache    sync.Map
	brokerDeltas             reconciliation.BrokerPendingDeltaReader
	cachedFraudExplainScorer fraud.Scorer
	fraudExplainScorerErr    error
	fraudExplainScorerMutex  sync.Mutex
	tcpControl               TCPControlPublisher
	nodeMetrics              *nodeadmin.MetricsWorker
	scoringWeights           *nodeadmin.ScoringWeightsStore
	leaseWorker              *shardadmin.OperationLeaseWorker
	pgFencing                *pgfailover.FencingGate
	globalSpend              *reconciliation.GlobalSpendReconciler
	rtbBidShadeSim           rtbadmin.BidShadeSimulator
	cloudflare               platformadmin.CloudflareAPI
	reputation               *domainhealth.ReputationChecker
	shard0Mu                 sync.Mutex
	reportJobRunner          *reportjob.ReportJobRunner
	landerStore              *landerhost.Store
	campaignRuntime          *runtime.Runtime
	campaignWorker           *campaignworker.Worker
	wizardStore              *wizard.WizardStore
	brandStore               *brand.Store
	supplyStore              *supply.Store
	flowStore                *flow.Store
	platformStore            *platformadmin.Store
	marginGuardStore         *marginguard.Store
	settingsStore            *settingsadmin.Store
	privacyStore             *privacyadmin.Store
	notifierAPI              notify.NotifierAPI
}

func (s *Service) SetRtbBidShadeSimulator(sim rtbadmin.BidShadeSimulator) {
	if s != nil {
		s.rtbBidShadeSim = sim
	}
}

func (s *Service) SetNotifier(api notify.NotifierAPI) {
	if s != nil {
		s.notifierAPI = api
	}
}

func (s *Service) StartBackgroundWorker(fn func()) {
	s.startWorker(fn)
}

func (s *Service) ClickHouseQuery() *database.ClickHouseQuery {
	if s == nil {
		return nil
	}
	return s.clickhouseQuery
}

func (s *Service) ClickHouseWrite() driver.Conn {
	if s == nil {
		return nil
	}
	return s.clickhouseWriteConn
}

func (s *Service) startWorker(fn func()) {
	s.workerMutex.Lock()
	if s.closed.Load() {
		s.workerMutex.Unlock()
		return
	}
	s.wg.Add(1)
	s.workerMutex.Unlock()

	go func() {
		defer s.wg.Done()
		fn()
	}()
}

func (s *Service) Close() {
	s.closed.Store(true)
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

func NewService(ctx context.Context, pool *pgxpool.Pool, redisShards []redis.UniversalClient, sharder domain.Sharder, cfg *config.Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	s := &Service{
		pool:        pool,
		redisShards: redisShards,
		sharder:     sharder,
		cfg:         cfg,
		ctx:         ctx,
		cancel:      cancel,
	}
	if cfg != nil {
		s.postgresGate = shardadmin.NewPostgresGate(cfg.DBTrackerMaxConns)
		s.cloudflare = platformadmin.NewCloudflareClient(string(cfg.Management.CloudflareAPIToken), cfg.Management.CloudflareAPIBase)
		s.initLanderStore()
	}
	if s = startBuiltinServiceWorkers(s, ctx, cfg, pool); s == nil {
		cancel()
		return nil
	}
	return s
}

func NewBareServiceForTest(t *testing.T, pool *pgxpool.Pool, redisShards []redis.UniversalClient, cfg *config.Config) *Service {
	t.Helper()
	if cfg == nil {
		cfg = &config.Config{}
	}
	shardCount := len(redisShards)
	if shardCount == 0 {
		shardCount = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	svc := &Service{
		redisShards: redisShards,
		sharder:     domain.NewStaticSlotSharder(shardCount),
		cfg:         cfg,
		ctx:         ctx,
		cancel:      cancel,
	}
	svc.SetPool(pool)
	if cfg.MultiRegionEnabled {
		svc.leaseWorker = NewOperationLeaseWorker(svc)
	}
	t.Cleanup(func() {
		cancel()
		svc.Close()
	})
	return svc
}

func NewRedisHostForOutboxTest(redisShards []redis.UniversalClient) *Service {
	shardCount := len(redisShards)
	if shardCount == 0 {
		shardCount = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		redisShards: redisShards,
		sharder:     domain.NewStaticSlotSharder(shardCount),
		cfg:         &config.Config{},
		ctx:         ctx,
		cancel:      cancel,
	}
}

var ErrClickHouseNotConfigured = campaign.ErrClickHouseNotConfigured

const campaignExportVersion = campaign.CampaignExportVersion

var errCampaignWizardIncomplete = campaign.ErrCampaignWizardIncomplete

func (s *Service) AssignCampaignFlow(ctx context.Context, campaignID, flowID uuid.UUID) error {
	return flow.AssignCampaignFlow(ctx, s.pool, s, campaignID, flowID)
}

func (s *Service) campaignFlowID(ctx context.Context, campaignID uuid.UUID) (string, error) {
	return flow.CampaignFlowID(ctx, s.pool, campaignID)
}
