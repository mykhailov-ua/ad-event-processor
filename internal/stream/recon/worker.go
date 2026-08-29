package recon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type EventBudgetChecker interface {
	Check(ctx context.Context, evt *domain.Event) error
}

type ReconciliationWorker struct {
	postgresConn   PostgresConn
	clickhouseConn ClickHouseConn
	repo           domain.CampaignRepository
	driftLimit     float64
	lag            time.Duration
	interval       time.Duration
}

func NewReconciliationWorker(
	postgresConn PostgresConn,
	clickhouseConn ClickHouseConn,
	repo domain.CampaignRepository,
	driftLimit float64,
	lag time.Duration,
	interval time.Duration,
) *ReconciliationWorker {
	return &ReconciliationWorker{
		postgresConn:   postgresConn,
		clickhouseConn: clickhouseConn,
		repo:           repo,
		driftLimit:     driftLimit,
		lag:            lag,
		interval:       interval,
	}
}

func (rw *ReconciliationWorker) Reconcile(ctx context.Context) error {
	campaigns, err := rw.repo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("reconciliation failed to list active campaigns: %w", err)
	}

	if len(campaigns) == 0 {
		return nil
	}

	until := time.Now().Add(-rw.lag)
	clickhouseSpends, err := rw.clickhouseConn.QueryAggregatedSpend(ctx, until)
	if err != nil {
		return fmt.Errorf("reconciliation failed to query ClickHouse aggregates: %w", err)
	}

	campaignIDs := make([]uuid.UUID, len(campaigns))
	for i, c := range campaigns {
		campaignIDs[i] = c.ID
	}
	postgresSpends, err := rw.postgresConn.GetCampaignSpends(ctx, campaignIDs)
	if err != nil {
		return fmt.Errorf("reconciliation failed to batch load Postgres spends: %w", err)
	}

	for _, c := range campaigns {
		postgresSpend := postgresSpends[c.ID]

		clickhouseSpend := clickhouseSpends[c.ID]

		var drift float64
		if postgresSpend > 0 {
			drift = math.Abs(float64(postgresSpend-clickhouseSpend)) / float64(postgresSpend)
		} else if clickhouseSpend > 0 {
			drift = 1.0
		}

		metrics.DataDriftRatio.WithLabelValues(c.ID.String()).Set(drift)

		if drift > rw.driftLimit {
			slog.Warn("Reconciliation: CRITICAL DATA DRIFT DETECTED",
				"campaign_id", c.ID,
				"postgres_spend", postgresSpend,
				"clickhouse_spend", clickhouseSpend,
				"drift_ratio", drift,
				"limit", rw.driftLimit,
			)
		} else {
			slog.Info("Reconciliation: campaign balances within normal drift limits",
				"campaign_id", c.ID,
				"postgres_spend", postgresSpend,
				"clickhouse_spend", clickhouseSpend,
				"drift_ratio", drift,
			)
		}
	}

	return nil
}

func (rw *ReconciliationWorker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(rw.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := rw.Reconcile(ctx); err != nil {
					slog.Error("Reconciliation: loop execution error", "error", err)
				}
			}
		}
	}()
}

func Count(list string) int {
	list = strings.TrimSpace(list)
	if list == "" {
		return 0
	}
	seen := make(map[int]struct{})
	for part := range strings.SplitSeq(list, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if i := strings.Index(part, "-"); i >= 0 {
			lo, err1 := strconv.Atoi(strings.TrimSpace(part[:i]))
			hi, err2 := strconv.Atoi(strings.TrimSpace(part[i+1:]))
			if err1 != nil || err2 != nil || hi < lo {
				continue
			}
			for cpu := lo; cpu <= hi; cpu++ {
				seen[cpu] = struct{}{}
			}
			continue
		}
		cpu, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		seen[cpu] = struct{}{}
	}
	return len(seen)
}

func EffectiveCount() (int, error) {
	paths := []string{
		"/sys/fs/cgroup/cpuset.cpus.effective",
		"/sys/fs/cgroup/cpuset.cpus",
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		n := Count(strings.TrimSpace(string(data)))
		if n > 0 {
			return n, nil
		}
	}
	return fromProcStatus()
}

func fromProcStatus() (int, error) {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "Cpus_allowed_list:") {
			continue
		}
		list := strings.TrimSpace(strings.TrimPrefix(line, "Cpus_allowed_list:"))
		n := Count(list)
		if n > 0 {
			return n, nil
		}
		break
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, os.ErrNotExist
}

func ApplyRuntimeAutotune() {
	applyGOMAXPROCS()
	if _, ok := os.LookupEnv("GOMEMLIMIT"); !ok {
		if mem, err := systemMemoryBytes(); err == nil && mem > 0 {
			limit := int64(float64(mem) * 0.9)
			debug.SetMemoryLimit(limit)
		}
	}
}

func DefaultMaxWorkers() int {
	if _, ok := os.LookupEnv("MAX_WORKERS"); ok {
		return 0
	}
	n := runtime.NumCPU()
	if n < 1 {
		n = 1
	}
	return n
}

func applyGOMAXPROCS() {
	if _, ok := os.LookupEnv("GOMAXPROCS"); ok {
		return
	}
	host := runtime.NumCPU()
	if host < 1 {
		host = 1
	}
	if n, err := EffectiveCount(); err == nil && n > 0 && n < host {
		runtime.GOMAXPROCS(n)
		return
	}
	if s := strings.TrimSpace(os.Getenv("TRACKER_CPUSET")); s != "" {
		if n := Count(s); n > 0 {
			runtime.GOMAXPROCS(n)
		}
	}
}

func systemMemoryBytes() (uint64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			break
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, err
		}
		return kb * 1024, nil
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, os.ErrNotExist
}

type Snapshot struct {
	CheckpointTime time.Time           `json:"checkpoint_time"`
	CampaignSpends map[uuid.UUID]int64 `json:"campaign_spends"`
}

type ClickHouseConn interface {
	QueryEventsSince(ctx context.Context, since time.Time) ([]*domain.Event, error)
	QueryAggregatedSpend(ctx context.Context, until time.Time) (map[uuid.UUID]int64, error)
}

type PostgresConn interface {
	UpdateCampaignSpend(ctx context.Context, campaignID uuid.UUID, currentSpend int64) error
	GetCampaignBudgetLimit(ctx context.Context, campaignID uuid.UUID) (int64, error)
	GetCampaignSpend(ctx context.Context, campaignID uuid.UUID) (int64, error)
	GetCampaignSpends(ctx context.Context, campaignIDs []uuid.UUID) (map[uuid.UUID]int64, error)
	MarkEventIdempotent(ctx context.Context, clickID string) (bool, error)
}

type SnapshotReplicator struct {
	mu             sync.RWMutex
	postgresConn   PostgresConn
	clickhouseConn ClickHouseConn
	redisShards    []redis.UniversalClient
	sharder        domain.Sharder
	clickCharge    int64
	impCharge      int64
}

func NewSnapshotReplicator(
	postgresConn PostgresConn,
	clickhouseConn ClickHouseConn,
	redisShards []redis.UniversalClient,
	sharder domain.Sharder,
	clickCharge, impCharge int64,
) *SnapshotReplicator {
	return &SnapshotReplicator{
		postgresConn:   postgresConn,
		clickhouseConn: clickhouseConn,
		redisShards:    redisShards,
		sharder:        sharder,
		clickCharge:    clickCharge,
		impCharge:      impCharge,
	}
}

func (sr *SnapshotReplicator) CreateSnapshot(ctx context.Context, until time.Time) ([]byte, error) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	spends, err := sr.clickhouseConn.QueryAggregatedSpend(ctx, until)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch aggregates from ClickHouse: %w", err)
	}

	snap := &Snapshot{
		CheckpointTime: until,
		CampaignSpends: spends,
	}

	data, err := json.Marshal(snap)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize snapshot: %w", err)
	}

	return data, nil
}

func (sr *SnapshotReplicator) RestoreSnapshot(ctx context.Context, snapshotData []byte) (*Snapshot, error) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	var snap Snapshot
	if err := json.Unmarshal(snapshotData, &snap); err != nil {
		return nil, fmt.Errorf("failed to parse snapshot data: %w", err)
	}

	for campID, spend := range snap.CampaignSpends {

		if err := sr.postgresConn.UpdateCampaignSpend(ctx, campID, spend); err != nil {
			return nil, fmt.Errorf("failed to update postgres campaign spend for %s: %w", campID, err)
		}

		limit, err := sr.postgresConn.GetCampaignBudgetLimit(ctx, campID)
		if err != nil {
			return nil, fmt.Errorf("failed to get campaign limit for %s: %w", campID, err)
		}

		remaining := limit - spend
		if remaining < 0 {
			remaining = 0
		}

		budgetKey := fmt.Sprintf("budget:campaign:%s", campID)
		shardIdx := sr.sharder.GetShard(campID)
		redisClient := sr.redisShards[shardIdx%len(sr.redisShards)]

		if err := redisClient.Set(ctx, budgetKey, remaining, 24*time.Hour).Err(); err != nil {
			return nil, fmt.Errorf("failed to seed redis budget for %s: %w", campID, err)
		}
	}

	return &snap, nil
}

func (sr *SnapshotReplicator) ReplayTelemetrySince(ctx context.Context, since time.Time, f EventBudgetChecker) (int, error) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	events, err := sr.clickhouseConn.QueryEventsSince(ctx, since)
	if err != nil {
		return 0, fmt.Errorf("failed to query raw telemetry since %s: %w", since, err)
	}

	replayedCount := 0
	for _, e := range events {

		isNew, err := sr.postgresConn.MarkEventIdempotent(ctx, e.ClickID)
		if err != nil {
			return replayedCount, fmt.Errorf("failed to execute idempotency check for %s: %w", e.ClickID, err)
		}
		if !isNew {
			continue
		}

		err = f.Check(ctx, e)
		if err != nil {

			if errors.Is(err, filter.ErrBudgetExhausted) {
				continue
			}
			return replayedCount, fmt.Errorf("failed to replay event %s: %w", e.ClickID, err)
		}

		charge := sr.clickCharge
		if e.Type == "impression" {
			charge = sr.impCharge
		}
		currentSpend, err := sr.postgresConn.GetCampaignSpend(ctx, e.CampaignID)
		if err == nil {
			_ = sr.postgresConn.UpdateCampaignSpend(ctx, e.CampaignID, currentSpend+charge)
		}

		replayedCount++
	}

	return replayedCount, nil
}
