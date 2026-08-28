package controlplane

import (
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/nodeadmin"
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	_ nodeadmin.ScorerHost  = (*Service)(nil)
	_ nodeadmin.MetricsHost = (*Service)(nil)
)

func (s *Service) ScorerConfig() nodeadmin.ScorerConfig {
	if s == nil || s.cfg == nil {
		return nodeadmin.DefaultScorerConfig()
	}
	return nodeadmin.ScorerConfigFrom(s.cfg)
}

func (s *Service) RegionCode() int16 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return int16(s.cfg.RegionCode)
}

func (s *Service) MultiRegionGlobal() bool {
	return s != nil && s.cfg != nil && s.cfg.MultiRegionGlobal()
}

func (s *Service) UDPSyncInterval() time.Duration {
	if s == nil || s.cfg == nil || s.cfg.UDPSyncIntervalMs <= 0 {
		return 0
	}
	return time.Duration(s.cfg.UDPSyncIntervalMs) * time.Millisecond
}

func (s *Service) ScoringMetricsForRole(role string) []nodeadmin.ScoringMetricDef {
	if s != nil && s.scoringWeights != nil {
		return s.scoringWeights.MetricsForRole(role)
	}
	return nodeadmin.MetricsForRole(role)
}

func (s *Service) NodeIdentity() (nodeID, role string, region int16) {
	nodeID, _ = os.Hostname()
	role = "management"
	if s != nil && s.cfg != nil {
		if s.cfg.NodeID != "" {
			nodeID = s.cfg.NodeID
		}
		if s.cfg.NodeRole != "" {
			role = s.cfg.NodeRole
		}
		region = int16(s.cfg.RegionCode)
	}
	return nodeID, role, region
}

func NewNodeCapacityScorer(svc *Service) *nodeadmin.NodeCapacityScorer {
	return nodeadmin.NewNodeCapacityScorer(svc)
}

func NewGlobalRegionTrafficScorer(svc *Service) *nodeadmin.GlobalRegionTrafficScorer {
	return nodeadmin.NewGlobalRegionTrafficScorer(svc)
}

func NewNodeCapacityScorerWorker(svc *Service) *nodeadmin.CapacityScorerWorker {
	return nodeadmin.NewCapacityScorerWorker(svc)
}

func NewGlobalRegionTrafficScorerWorker(svc *Service) *nodeadmin.GlobalTrafficScorerWorker {
	return nodeadmin.NewGlobalTrafficScorerWorker(svc)
}

func NewNodeMetricsWorker(svc *Service) *nodeadmin.MetricsWorker {
	return nodeadmin.NewMetricsWorker(svc)
}

func NewNodeMetricsSnapshotWorker(svc *Service) *nodeadmin.MetricsSnapshotWorker {
	return nodeadmin.NewMetricsSnapshotWorker(svc)
}

func NewScoringWeightsStore(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) (*nodeadmin.ScoringWeightsStore, error) {
	return nodeadmin.NewScoringWeightsStore(ctx, pool, cfg)
}

func ValidateScoringWeightsConfig(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) error {
	return nodeadmin.ValidateScoringWeightsConfig(ctx, pool, cfg)
}

type (
	ScorerConfig              = nodeadmin.ScorerConfig
	ScoringMetricDef          = nodeadmin.ScoringMetricDef
	ScoringWeightsStore       = nodeadmin.ScoringWeightsStore
	NodeCapacityScorer        = nodeadmin.NodeCapacityScorer
	GlobalRegionTrafficScorer = nodeadmin.GlobalRegionTrafficScorer
	NodeScoreInput            = nodeadmin.NodeScoreInput
	NodeScoreResult           = nodeadmin.NodeScoreResult
	NodeScoreState            = nodeadmin.NodeScoreState
	BucketPoint               = nodeadmin.BucketPoint
	RegionDialInput           = nodeadmin.RegionDialInput
	RegionDialResult          = nodeadmin.RegionDialResult
	MetricKind                = nodeadmin.MetricKind
	ScorePhase                = nodeadmin.ScorePhase
)

const (
	RoleTracker     = nodeadmin.RoleTracker
	RoleRegionProxy = nodeadmin.RoleRegionProxy
	RoleProcessor   = nodeadmin.RoleProcessor

	ProvenanceOwnWindow           = nodeadmin.ProvenanceOwnWindow
	ProvenanceNeighborMedian      = nodeadmin.ProvenanceNeighborMedian
	ProvenanceHistoricalDaily     = nodeadmin.ProvenanceHistoricalDaily
	ProvenanceConservativeDefault = nodeadmin.ProvenanceConservativeDefault

	MetricLatency     = nodeadmin.MetricLatency
	MetricUtilization = nodeadmin.MetricUtilization
	MetricRate        = nodeadmin.MetricRate
	MetricCounter     = nodeadmin.MetricCounter
)

func DefaultScorerConfig() ScorerConfig { return nodeadmin.DefaultScorerConfig() }

func ScorerConfigFrom(cfg *config.Config) ScorerConfig { return nodeadmin.ScorerConfigFrom(cfg) }

func ScoreNode(in NodeScoreInput, cfg ScorerConfig) NodeScoreResult {
	return nodeadmin.ScoreNode(in, cfg)
}

func ScoreNodes(inputs []NodeScoreInput, cfg ScorerConfig) []NodeScoreResult {
	return nodeadmin.ScoreNodes(inputs, cfg)
}

func ComputeCapacityScoreFromValues(role string, values map[string]float64, defs []ScoringMetricDef) float64 {
	return nodeadmin.ComputeCapacityScoreFromValues(role, values, defs)
}

func MetricsForRole(role string) []ScoringMetricDef { return nodeadmin.MetricsForRole(role) }

func DefaultTrackerMetrics() []ScoringMetricDef { return nodeadmin.DefaultTrackerMetrics() }

func DefaultRegionProxyMetrics() []ScoringMetricDef { return nodeadmin.DefaultRegionProxyMetrics() }

func ComputeRegionDialResults(inputs []RegionDialInput, cfg ScorerConfig) []RegionDialResult {
	return nodeadmin.ComputeRegionDialResults(inputs, cfg)
}

func NormalizePeerWeights(weights []float64, minW, maxW float64) []float64 {
	return nodeadmin.NormalizePeerWeights(weights, minW, maxW)
}

func ApplyHardSignals(weight float64, diskDegraded, budgetInvariantFail bool) float64 {
	return nodeadmin.ApplyHardSignals(weight, diskDegraded, budgetInvariantFail)
}

func SortNodeIDs(ids []string) []string { return nodeadmin.SortNodeIDs(ids) }

func HistoricalSnapshotDay(now time.Time) time.Time { return nodeadmin.HistoricalSnapshotDay(now) }

func LookupHistoricalDaily(ctx context.Context, pool *pgxpool.Pool, region int16, role, metric string, kind MetricKind, now time.Time) (*float64, error) {
	return nodeadmin.LookupHistoricalDaily(ctx, pool, region, role, metric, kind, now)
}
