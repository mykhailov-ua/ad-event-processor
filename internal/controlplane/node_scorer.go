package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ProvenanceOwnWindow           = "own_window"
	ProvenanceNeighborMedian      = "neighbor_median"
	ProvenanceHistoricalDaily     = "historical_daily"
	ProvenanceConservativeDefault = "conservative_default"
)

const (
	defaultScoreWindowMin    = 15
	defaultScoreMinSamples   = 30
	defaultScoreEMAAlpha     = 0.3
	defaultMinDrainEpochs    = 3
	defaultDrainStep         = 0.10
	defaultBoostStep         = 0.05
	defaultWeightMin         = 0.05
	defaultWeightMax         = 0.95
	defaultWeightMaxCold     = 0.25
	defaultConservativeScore = 0.5
	defaultDrainThreshold    = 0.3
	defaultBoostThreshold    = 0.8
	defaultNodeWarmupSec     = 300
	maxScrapeMissEpochs      = 2
)

const (
	defaultMaxWeightDeltaPerEpoch = 0.10
	defaultHistoricalDrainStep    = 0.02
)

const (
	scoringWeightsSettingKey = "scoring_weights_json"
	weightSumEpsilon         = 1e-6
)

const scoringWeightsReloadInterval = 60 * time.Second

type MetricKind int

const (
	MetricLatency MetricKind = iota
	MetricUtilization
	MetricRate
	MetricCounter
)

type ScorePhase string

const (
	PhaseColdStart    ScorePhase = "A"
	PhaseOwnWindow    ScorePhase = "B"
	PhaseNeighbor     ScorePhase = "C"
	PhaseHistorical   ScorePhase = "D"
	PhaseConservative ScorePhase = "E"
)

type BucketPoint struct {
	P50, P99, Mean         float64
	SampleCount            int64
	Numerator, Denominator float64
}

type ScorerConfig struct {
	WindowMin         int
	MinSamples        int
	EMAAlpha          float64
	MinDrainEpochs    int
	DrainStep         float64
	BoostStep         float64
	WeightMin         float64
	WeightMax         float64
	WeightMaxCold     float64
	DrainThreshold    float64
	BoostThreshold    float64
	ConservativeScore float64
	NodeWarmupSec     int
}

type ScoringWeightsByRole map[string]map[string]float64

type ScoringWeightsStore struct {
	defs atomic.Pointer[map[string][]ScoringMetricDef]
}

type NodeScoreState struct {
	EMAScore    float64
	DrainEpochs int
}

type NodeScoreInput struct {
	Uptime           time.Duration
	Buckets          []BucketPoint
	Kind             MetricKind
	ScrapeMissEpochs int
	NeighborValues   []float64
	HistoricalValue  *float64
	PreviousWeight   float64
	State            NodeScoreState
}

type NodeScoreResult struct {
	RawValue     float64
	Score        float64
	Provenance   string
	Weight       float64
	DrainEpochs  int
	ColdStart    bool
	WeightFrozen bool
}

type RegionDialInput struct {
	RegionCode int16
	Nodes      []db.NodeCapacityScore
	PrevWeight float64
	State      NodeScoreState
}

type RegionDialResult struct {
	RegionCode int16
	Score      float64
	Weight     float64
	Provenance string
	State      NodeScoreState
}

func DefaultScorerConfig() ScorerConfig {
	return ScorerConfig{
		WindowMin:         defaultScoreWindowMin,
		MinSamples:        defaultScoreMinSamples,
		EMAAlpha:          defaultScoreEMAAlpha,
		MinDrainEpochs:    defaultMinDrainEpochs,
		DrainStep:         defaultDrainStep,
		BoostStep:         defaultBoostStep,
		WeightMin:         defaultWeightMin,
		WeightMax:         defaultWeightMax,
		WeightMaxCold:     defaultWeightMaxCold,
		DrainThreshold:    defaultDrainThreshold,
		BoostThreshold:    defaultBoostThreshold,
		ConservativeScore: defaultConservativeScore,
		NodeWarmupSec:     defaultNodeWarmupSec,
	}
}

func aggregateWindow(buckets []BucketPoint, kind MetricKind) (float64, bool) {
	if len(buckets) == 0 {
		return 0, false
	}
	switch kind {
	case MetricLatency:
		maxP99 := buckets[0].P99
		for _, b := range buckets[1:] {
			if b.P99 > maxP99 {
				maxP99 = b.P99
			}
		}
		return maxP99, true
	case MetricUtilization:
		var sum float64
		var n int64
		for _, b := range buckets {
			if b.SampleCount > 0 {
				sum += b.Mean
				n++
			}
		}
		if n == 0 {
			return 0, false
		}
		return sum / float64(n), true
	case MetricRate:
		var num, den float64
		for _, b := range buckets {
			if b.Denominator > 0 {
				num += b.Numerator
				den += b.Denominator
				continue
			}
			if b.SampleCount > 0 {
				num += b.Mean * float64(b.SampleCount)
				den += float64(b.SampleCount)
			}
		}
		if den == 0 {
			return 0, false
		}
		return num / den, true
	case MetricCounter:
		maxVal := buckets[0].Mean
		for _, b := range buckets[1:] {
			if b.Mean > maxVal {
				maxVal = b.Mean
			}
		}
		return maxVal, true
	default:
		return 0, false
	}
}

func ScorerConfigFrom(cfg *config.Config) ScorerConfig {
	c := DefaultScorerConfig()
	if cfg == nil {
		return c
	}
	if cfg.NodeScoreWindowMin > 0 {
		c.WindowMin = cfg.NodeScoreWindowMin
	}
	if cfg.NodeScoreMinSamples > 0 {
		c.MinSamples = cfg.NodeScoreMinSamples
	}
	if cfg.NodeWarmupSec > 0 {
		c.NodeWarmupSec = cfg.NodeWarmupSec
	}
	return c
}

func ScoreNode(in NodeScoreInput, cfg ScorerConfig) NodeScoreResult {
	if cfg.WindowMin <= 0 {
		cfg.WindowMin = defaultScoreWindowMin
	}
	if cfg.MinSamples <= 0 {
		cfg.MinSamples = defaultScoreMinSamples
	}
	if cfg.EMAAlpha <= 0 {
		cfg.EMAAlpha = defaultScoreEMAAlpha
	}
	if cfg.MinDrainEpochs <= 0 {
		cfg.MinDrainEpochs = defaultMinDrainEpochs
	}
	if cfg.ConservativeScore <= 0 {
		cfg.ConservativeScore = defaultConservativeScore
	}

	warmup := time.Duration(cfg.WindowMin) * time.Minute
	coldStart := in.Uptime < warmup
	ownSamples := totalBucketSamples(in.Buckets)
	ownOK := ownSamples >= int64(cfg.MinSamples) && in.ScrapeMissEpochs <= maxScrapeMissEpochs

	var (
		raw        float64
		provenance string
		ok         bool
	)

	switch {
	case coldStart:
		raw, ok = neighborMedian(in.NeighborValues)
		provenance = ProvenanceNeighborMedian
	case ownOK:
		raw, ok = aggregateWindow(in.Buckets, in.Kind)
		provenance = ProvenanceOwnWindow
	case len(validNeighborValues(in.NeighborValues)) >= 2:
		raw, ok = neighborMedian(in.NeighborValues)
		provenance = ProvenanceNeighborMedian
	case in.HistoricalValue != nil:
		raw = *in.HistoricalValue
		ok = true
		provenance = ProvenanceHistoricalDaily
	default:
		raw = cfg.ConservativeScore
		ok = true
		provenance = ProvenanceConservativeDefault
	}

	if !ok {
		raw = cfg.ConservativeScore
		provenance = ProvenanceConservativeDefault
	}

	if provenance != ProvenanceOwnWindow {
		metrics.ControlScoreFallbackTotal.WithLabelValues(provenance).Inc()
	}

	score := smoothScore(in.State.EMAScore, raw, cfg.EMAAlpha)
	warmupGrace := nodeWarmupGrace(in.Uptime, cfg)
	weight, drainEpochs, frozen := applyWeightHysteresis(in.PreviousWeight, score, provenance, coldStart, ownOK, in.State.DrainEpochs, warmupGrace, cfg)

	return NodeScoreResult{
		RawValue:     raw,
		Score:        score,
		Provenance:   provenance,
		Weight:       weight,
		DrainEpochs:  drainEpochs,
		ColdStart:    coldStart,
		WeightFrozen: frozen,
	}
}

func ScoreNodes(inputs []NodeScoreInput, cfg ScorerConfig) []NodeScoreResult {
	out := make([]NodeScoreResult, len(inputs))
	for i := range inputs {
		out[i] = ScoreNode(inputs[i], cfg)
	}
	return out
}

func ResolveScorePhase(in NodeScoreInput, cfg ScorerConfig) ScorePhase {
	if cfg.WindowMin <= 0 {
		cfg.WindowMin = defaultScoreWindowMin
	}
	if cfg.MinSamples <= 0 {
		cfg.MinSamples = defaultScoreMinSamples
	}
	warmup := time.Duration(cfg.WindowMin) * time.Minute
	if in.Uptime < warmup {
		return PhaseColdStart
	}
	ownSamples := totalBucketSamples(in.Buckets)
	ownOK := ownSamples >= int64(cfg.MinSamples) && in.ScrapeMissEpochs <= maxScrapeMissEpochs
	if ownOK {
		return PhaseOwnWindow
	}
	if len(validNeighborValues(in.NeighborValues)) >= 2 {
		return PhaseNeighbor
	}
	if in.HistoricalValue != nil {
		return PhaseHistorical
	}
	return PhaseConservative
}

func ParseScoringWeightsJSON(raw string) (ScoringWeightsByRole, error) {
	raw = trimJSON(raw)
	if raw == "" {
		return nil, nil
	}
	var parsed ScoringWeightsByRole
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("parse scoring weights json: %w", err)
	}
	if err := ValidateScoringWeightsByRole(parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func ValidateScoringWeightsByRole(byRole ScoringWeightsByRole) error {
	if len(byRole) == 0 {
		return nil
	}
	for role, weights := range byRole {
		if len(weights) == 0 {
			return fmt.Errorf("scoring weights role=%s: empty metric map", role)
		}
		if err := validateRoleMetricWeights(role, weights); err != nil {
			return err
		}
	}
	return nil
}

func RenormalizeMetricWeights(weights map[string]float64) map[string]float64 {
	if len(weights) == 0 {
		return weights
	}
	var sum float64
	for _, w := range weights {
		sum += w
	}
	if sum <= 0 {
		return weights
	}
	out := make(map[string]float64, len(weights))
	for name, w := range weights {
		out[name] = w / sum
	}
	return out
}

func ApplyScoringWeights(defs []ScoringMetricDef, weights map[string]float64) []ScoringMetricDef {
	out := make([]ScoringMetricDef, len(defs))
	copy(out, defs)
	if len(weights) == 0 {
		return out
	}
	for i := range out {
		if w, ok := weights[out[i].Name]; ok {
			out[i].Weight = w
		}
	}
	return out
}

func BuildScoringMetricDefsByRole(byRole ScoringWeightsByRole) map[string][]ScoringMetricDef {
	roles := []string{RoleTracker, RoleRegionProxy, RoleProcessor}
	out := make(map[string][]ScoringMetricDef, len(roles))
	for _, role := range roles {
		defs := MetricsForRole(role)
		if weights, ok := byRole[role]; ok {
			defs = ApplyScoringWeights(defs, weights)
		}
		out[role] = defs
	}
	return out
}

func AggregateRegionDialScore(nodes []db.NodeCapacityScore) (score float64, provenance string, ok bool) {
	if len(nodes) == 0 {
		return defaultConservativeScore, ProvenanceConservativeDefault, false
	}
	var scoreSum, weightSum float64
	provs := make([]string, 0, len(nodes))
	for _, n := range nodes {
		w := n.Weight
		if w <= 0 {
			w = 1
		}
		scoreSum += n.Score * w
		weightSum += w
		provs = append(provs, n.Provenance)
	}
	if weightSum <= 0 {
		for _, n := range nodes {
			scoreSum += n.Score
		}
		return scoreSum / float64(len(nodes)), DominantProvenance(provs), true
	}
	return scoreSum / weightSum, DominantProvenance(provs), true
}

func ComputeRegionDialResults(inputs []RegionDialInput, cfg ScorerConfig) []RegionDialResult {
	if len(inputs) == 0 {
		return nil
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].RegionCode < inputs[j].RegionCode })

	warmUptime := time.Duration(cfg.WindowMin+1) * time.Minute
	rawScores := make([]float64, len(inputs))
	rawWeights := make([]float64, len(inputs))
	provenances := make([]string, len(inputs))
	states := make([]NodeScoreState, len(inputs))
	for i, in := range inputs {
		score, prov, _ := AggregateRegionDialScore(in.Nodes)
		rawScores[i] = score
		provenances[i] = prov

		prev := in.PrevWeight
		if prev <= 0 {
			prev = 1.0 / float64(len(inputs))
		}
		weightRes := ScoreNode(NodeScoreInput{
			Uptime:         warmUptime,
			Kind:           MetricUtilization,
			Buckets:        []BucketPoint{{Mean: score, SampleCount: int64(cfg.MinSamples)}},
			PreviousWeight: prev,
			State:          in.State,
		}, cfg)
		rawWeights[i] = weightRes.Weight
		states[i] = NodeScoreState{EMAScore: weightRes.Score, DrainEpochs: weightRes.DrainEpochs}
	}

	normWeights := NormalizePeerWeights(rawWeights, cfg.WeightMin, cfg.WeightMax)
	out := make([]RegionDialResult, len(inputs))
	for i, in := range inputs {
		out[i] = RegionDialResult{
			RegionCode: in.RegionCode,
			Score:      rawScores[i],
			Weight:     normWeights[i],
			Provenance: provenances[i],
			State:      states[i],
		}
	}
	return out
}

func HistoricalSnapshotDay(now time.Time) time.Time {
	d := now.UTC().AddDate(0, 0, -1)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}

func LookupHistoricalDaily(
	ctx context.Context,
	pool *pgxpool.Pool,
	region int16,
	role, metric string,
	kind MetricKind,
	now time.Time,
) (*float64, error) {
	if pool == nil {
		return nil, nil
	}
	day := HistoricalSnapshotDay(now)
	q := db.New(pool)
	row, err := q.GetNodeMetricDailySnapshot(ctx, db.GetNodeMetricDailySnapshotParams{
		Day:        pgtype.Date{Time: day, Valid: true},
		RegionCode: region,
		Role:       role,
		Metric:     metric,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	raw, ok := historicalRawFromSnapshot(row, kind)
	if !ok {
		return nil, nil
	}
	return &raw, nil
}

func ValidateScoringWeightsConfig(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) error {
	if cfg == nil || !cfg.MultiRegionEnabled {
		return nil
	}
	_, err := loadScoringWeightsFromSources(ctx, pool, cfg)
	if err != nil {
		return fmt.Errorf("validate scoring weights config: %w", err)
	}
	return nil
}

func NewScoringWeightsStore(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) (*ScoringWeightsStore, error) {
	store := &ScoringWeightsStore{}
	if err := store.reload(ctx, pool, cfg); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *ScoringWeightsStore) Start(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) {
	if s == nil {
		return
	}
	ticker := time.NewTicker(scoringWeightsReloadInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.reload(ctx, pool, cfg); err != nil {
				slog.Warn("scoring weights reload failed", "error", err)
			}
		}
	}
}

func (s *ScoringWeightsStore) MetricsForRole(role string) []ScoringMetricDef {
	if s == nil {
		return MetricsForRole(role)
	}
	m := s.defs.Load()
	if m == nil {
		return MetricsForRole(role)
	}
	if defs, ok := (*m)[role]; ok {
		return defs
	}
	return MetricsForRole(role)
}

func (s *ScoringWeightsStore) reload(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) error {
	byRole, err := loadScoringWeightsFromSources(ctx, pool, cfg)
	if err != nil {
		return err
	}
	defs := BuildScoringMetricDefsByRole(byRole)
	s.defs.Store(&defs)
	return nil
}

func totalBucketSamples(buckets []BucketPoint) int64 {
	var n int64
	for _, b := range buckets {
		n += b.SampleCount
	}
	return n
}

func validNeighborValues(values []float64) []float64 {
	out := make([]float64, 0, len(values))
	for _, v := range values {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			out = append(out, v)
		}
	}
	return out
}

func neighborMedian(values []float64) (float64, bool) {
	vals := validNeighborValues(values)
	if len(vals) == 0 {
		return 0, false
	}
	sort.Float64s(vals)
	mid := len(vals) / 2
	if len(vals)%2 == 1 {
		return vals[mid], true
	}
	return (vals[mid-1] + vals[mid]) / 2, true
}

func smoothScore(prevEMA, raw, alpha float64) float64 {
	if prevEMA <= 0 {
		return raw
	}
	return alpha*raw + (1-alpha)*prevEMA
}

func applyWeightHysteresis(prevWeight, score float64, provenance string, coldStart, ownOK bool, drainEpochs int, warmupGrace bool, cfg ScorerConfig) (weight float64, outDrain int, frozen bool) {
	weight = prevWeight
	if prevWeight <= 0 {
		weight = 1.0 / float64(maxInt(1, cfg.MinSamples))
		if weight > cfg.WeightMax {
			weight = cfg.WeightMax
		}
	}

	if provenance == ProvenanceConservativeDefault {
		return prevWeight, 0, true
	}

	if coldStart && provenance == ProvenanceNeighborMedian {
		if weight > cfg.WeightMaxCold {
			weight = cfg.WeightMaxCold
		}
		return weight, 0, false
	}

	if !ownOK && provenance == ProvenanceNeighborMedian {
		return weight, 0, false
	}

	if warmupGrace && provenance == ProvenanceOwnWindow {
		if weight > cfg.WeightMaxCold {
			weight = cfg.WeightMaxCold
		}
		return weight, 0, false
	}

	if provenance == ProvenanceHistoricalDaily {
		weight = math.Max(cfg.WeightMin, weight-defaultHistoricalDrainStep)
		weight = clampWeightDelta(prevWeight, weight, defaultHistoricalDrainStep)
		return weight, 0, false
	}

	outDrain = drainEpochs
	if score < cfg.DrainThreshold {
		outDrain++
		if outDrain >= cfg.MinDrainEpochs {
			weight = math.Max(cfg.WeightMin, weight-cfg.DrainStep)
		}
	} else {
		outDrain = 0
	}

	if score > cfg.BoostThreshold && provenance == ProvenanceOwnWindow {
		weight = math.Min(cfg.WeightMax, weight+cfg.BoostStep)
	}
	weight = clampWeightDelta(prevWeight, weight, defaultMaxWeightDeltaPerEpoch)
	return weight, outDrain, false
}

func clampWeightDelta(prev, next, maxDelta float64) float64 {
	if maxDelta <= 0 {
		return next
	}
	delta := next - prev
	if delta > maxDelta {
		return prev + maxDelta
	}
	if delta < -maxDelta {
		return prev - maxDelta
	}
	return next
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func nodeWarmupGrace(uptime time.Duration, cfg ScorerConfig) bool {
	sec := cfg.NodeWarmupSec
	if sec <= 0 {
		sec = defaultNodeWarmupSec
	}
	return uptime < time.Duration(sec)*time.Second
}

func validateRoleMetricWeights(role string, weights map[string]float64) error {
	defs := MetricsForRole(role)
	if len(defs) == 0 {
		return fmt.Errorf("scoring weights role=%s: unknown role", role)
	}
	known := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		known[def.Name] = struct{}{}
	}
	var sum float64
	for name, w := range weights {
		if _, ok := known[name]; !ok {
			return fmt.Errorf("scoring weights role=%s: unknown metric %q", role, name)
		}
		if w < 0 || math.IsNaN(w) || math.IsInf(w, 0) {
			return fmt.Errorf("scoring weights role=%s metric=%s: invalid weight %v", role, name, w)
		}
		sum += w
	}
	for _, def := range defs {
		if _, ok := weights[def.Name]; !ok {
			return fmt.Errorf("scoring weights role=%s: missing metric %q", role, def.Name)
		}
	}
	if math.Abs(sum-1.0) > weightSumEpsilon {
		return fmt.Errorf("scoring weights role=%s: sum=%f want 1", role, sum)
	}
	return nil
}

func historicalRawFromSnapshot(row db.NodeMetricDailySnapshot, kind MetricKind) (float64, bool) {
	switch kind {
	case MetricLatency:
		if row.ValueP99.Valid {
			return row.ValueP99.Float64, true
		}
	case MetricUtilization, MetricRate, MetricCounter:
		if row.ValueMean.Valid {
			return row.ValueMean.Float64, true
		}
	default:
		if row.ValueMean.Valid {
			return row.ValueMean.Float64, true
		}
	}
	return 0, false
}

func loadScoringWeightsFromSources(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) (ScoringWeightsByRole, error) {
	raw := ""
	if cfg != nil {
		raw = trimJSON(cfg.ScoringWeightsJSON)
	}
	if raw == "" && pool != nil {
		q := db.New(pool)
		val, err := q.GetSystemSetting(ctx, scoringWeightsSettingKey)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("load scoring weights from system_settings: %w", err)
			}
		} else {
			raw = trimJSON(val)
		}
	}
	if raw == "" {
		return nil, nil
	}
	return ParseScoringWeightsJSON(raw)
}

func trimJSON(raw string) string {
	for len(raw) > 0 && (raw[0] == ' ' || raw[0] == '\n' || raw[0] == '\t' || raw[0] == '\r') {
		raw = raw[1:]
	}
	for len(raw) > 0 {
		last := raw[len(raw)-1]
		if last != ' ' && last != '\n' && last != '\t' && last != '\r' {
			break
		}
		raw = raw[:len(raw)-1]
	}
	return raw
}
