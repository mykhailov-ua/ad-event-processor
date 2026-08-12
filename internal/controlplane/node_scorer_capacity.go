package controlplane

import (
	"math"
	"sort"
)

const (
	RoleTracker     = "tracker"
	RoleRegionProxy = "region-proxy"
	RoleProcessor   = "processor"
)

const (
	MetricCPUUtil              = "cpu_util"
	MetricRAMUtil              = "ram_util"
	MetricDiskFsyncP99MS       = "disk_fsync_p99_ms"
	MetricDiskGateWaitP99MS    = "disk_gate_wait_p99_ms"
	MetricIngressP99MS         = "ingress_p99_ms"
	MetricFraudRejectRate      = "fraud_reject_rate"
	MetricIVTRate              = "ivt_rate"
	MetricBudgetInvariantDrift = "budget_invariant_drift"
	MetricStreamLagBytes       = "stream_lag_bytes"
	MetricKeygenQueueDepth     = "keygen_queue_depth"
	MetricDiskDegraded         = "disk_degraded"
	MetricBudgetInvariantFail  = "budget_invariant_fail"
)

const (
	defaultUtilCeil          = 0.90
	defaultDiskFsyncGoodMS   = 10.0
	defaultDiskFsyncBadMS    = 50.0
	defaultGateWaitGoodMS    = 5.0
	defaultGateWaitBadMS     = 30.0
	defaultIngressGoodMS     = 50.0
	defaultIngressBadMS      = 100.0
	defaultFraudRejectMax    = 0.05
	defaultIVTRateMax        = 0.03
	defaultBudgetDriftMax    = 1.0
	defaultStreamLagMaxBytes = 1_000_000.0
	defaultKeygenQueueMax    = 1000.0
	maxProxyKeygenPenalty    = 0.50
)

type ScoringMetricDef struct {
	Name   string
	Weight float64
	Kind   MetricKind
	Norm   normSpec
}

type normSpec struct {
	utilCeil   float64
	latGoodMS  float64
	latBadMS   float64
	rateMax    float64
	counterMax float64
}

func DefaultTrackerMetrics() []ScoringMetricDef {
	return []ScoringMetricDef{
		{MetricCPUUtil, 0.20, MetricUtilization, normSpec{utilCeil: defaultUtilCeil}},
		{MetricRAMUtil, 0.15, MetricUtilization, normSpec{utilCeil: defaultUtilCeil}},
		{MetricDiskFsyncP99MS, 0.15, MetricLatency, normSpec{latGoodMS: defaultDiskFsyncGoodMS, latBadMS: defaultDiskFsyncBadMS}},
		{MetricDiskGateWaitP99MS, 0.10, MetricLatency, normSpec{latGoodMS: defaultGateWaitGoodMS, latBadMS: defaultGateWaitBadMS}},
		{MetricIngressP99MS, 0.10, MetricLatency, normSpec{latGoodMS: defaultIngressGoodMS, latBadMS: defaultIngressBadMS}},
		{MetricFraudRejectRate, 0.10, MetricRate, normSpec{rateMax: defaultFraudRejectMax}},
		{MetricIVTRate, 0.08, MetricRate, normSpec{rateMax: defaultIVTRateMax}},
		{MetricBudgetInvariantDrift, 0.07, MetricRate, normSpec{rateMax: defaultBudgetDriftMax}},
		{MetricStreamLagBytes, 0.05, MetricCounter, normSpec{counterMax: defaultStreamLagMaxBytes}},
	}
}

func DefaultRegionProxyMetrics() []ScoringMetricDef {
	return []ScoringMetricDef{
		{MetricCPUUtil, 0.28, MetricUtilization, normSpec{utilCeil: defaultUtilCeil}},
		{MetricRAMUtil, 0.22, MetricUtilization, normSpec{utilCeil: defaultUtilCeil}},
		{MetricDiskFsyncP99MS, 0.28, MetricLatency, normSpec{latGoodMS: defaultDiskFsyncGoodMS, latBadMS: defaultDiskFsyncBadMS}},
		{MetricDiskGateWaitP99MS, 0.17, MetricLatency, normSpec{latGoodMS: defaultGateWaitGoodMS, latBadMS: defaultGateWaitBadMS}},
		{MetricStreamLagBytes, 0.05, MetricCounter, normSpec{counterMax: defaultStreamLagMaxBytes}},
	}
}

func DefaultProcessorMetrics() []ScoringMetricDef {
	return []ScoringMetricDef{
		{MetricCPUUtil, 0.30, MetricUtilization, normSpec{utilCeil: defaultUtilCeil}},
		{MetricRAMUtil, 0.20, MetricUtilization, normSpec{utilCeil: defaultUtilCeil}},
		{MetricDiskFsyncP99MS, 0.15, MetricLatency, normSpec{latGoodMS: defaultDiskFsyncGoodMS, latBadMS: defaultDiskFsyncBadMS}},
		{MetricFraudRejectRate, 0.15, MetricRate, normSpec{rateMax: defaultFraudRejectMax}},
		{MetricStreamLagBytes, 0.20, MetricCounter, normSpec{counterMax: defaultStreamLagMaxBytes}},
	}
}

func MetricsForRole(role string) []ScoringMetricDef {
	switch role {
	case RoleRegionProxy:
		return DefaultRegionProxyMetrics()
	case RoleProcessor:
		return DefaultProcessorMetrics()
	default:
		return DefaultTrackerMetrics()
	}
}

func NormalizeMetricHealth(raw float64, def ScoringMetricDef) float64 {
	if math.IsNaN(raw) || math.IsInf(raw, 0) {
		return 0
	}
	switch def.Kind {
	case MetricUtilization:
		ceil := def.Norm.utilCeil
		if ceil <= 0 {
			ceil = defaultUtilCeil
		}
		return clamp01(1 - raw/ceil)
	case MetricLatency:
		good, bad := def.Norm.latGoodMS, def.Norm.latBadMS
		if bad <= good {
			return 0
		}
		if raw <= good {
			return 1
		}
		if raw >= bad {
			return 0
		}
		return clamp01(1 - (raw-good)/(bad-good))
	case MetricRate:
		rateMax := def.Norm.rateMax
		if rateMax <= 0 {
			return 1
		}
		return clamp01(1 - raw/rateMax)
	case MetricCounter:
		counterMax := def.Norm.counterMax
		if counterMax <= 0 {
			return 1
		}
		return clamp01(1 - raw/counterMax)
	default:
		return clamp01(1 - raw)
	}
}

func ComputeCapacityScoreFromValues(role string, values map[string]float64, defs []ScoringMetricDef) float64 {
	var score, wsum float64
	for _, def := range defs {
		raw, ok := values[def.Name]
		if !ok {
			continue
		}
		health := NormalizeMetricHealth(raw, def)
		score += def.Weight * health
		wsum += def.Weight
	}
	if wsum == 0 {
		return defaultConservativeScore
	}
	out := score / wsum
	return applyProxyKeygenPenalty(role, values, out)
}

func applyProxyKeygenPenalty(role string, values map[string]float64, score float64) float64 {
	if role != RoleRegionProxy {
		return clamp01(score)
	}
	depth, ok := values[MetricKeygenQueueDepth]
	if !ok {
		return clamp01(score)
	}
	penalty := clamp01(depth/defaultKeygenQueueMax) * maxProxyKeygenPenalty
	return clamp01(score * (1 - penalty))
}

func ApplyHardSignals(weight float64, diskDegraded, budgetInvariantFail bool) float64 {
	if diskDegraded || budgetInvariantFail {
		return 0
	}
	return weight
}

func HardSignalsActive(values map[string]float64) (diskDegraded, budgetInvariantFail bool) {
	if v, ok := values[MetricDiskDegraded]; ok && v >= 1 {
		diskDegraded = true
	}
	if v, ok := values[MetricBudgetInvariantFail]; ok && v >= 1 {
		budgetInvariantFail = true
	}
	return diskDegraded, budgetInvariantFail
}

func NormalizePeerWeights(weights []float64, minW, maxW float64) []float64 {
	if len(weights) == 0 {
		return nil
	}
	out := make([]float64, len(weights))
	var sum float64
	for i, w := range weights {
		if w == 0 {
			out[i] = 0
			continue
		}
		out[i] = clamp(w, minW, maxW)
		sum += out[i]
	}
	if sum <= 0 {
		active := 0
		for _, w := range out {
			if w > 0 {
				active++
			}
		}
		if active == 0 {
			eq := 1.0 / float64(len(out))
			for i := range out {
				out[i] = eq
			}
			return out
		}
		eq := 1.0 / float64(active)
		for i := range out {
			if out[i] > 0 {
				out[i] = eq
			}
		}
		return out
	}
	for i := range out {
		if out[i] > 0 {
			out[i] /= sum
		}
	}
	return out
}

func MeanCapacityScore(scores []float64) float64 {
	vals := validNeighborValues(scores)
	if len(vals) == 0 {
		return defaultConservativeScore
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func clamp01(v float64) float64 {
	return clamp(v, 0, 1)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func DominantProvenance(provenances []string) string {
	if len(provenances) == 0 {
		return ProvenanceConservativeDefault
	}
	order := map[string]int{
		ProvenanceConservativeDefault: 0,
		ProvenanceHistoricalDaily:     1,
		ProvenanceNeighborMedian:      2,
		ProvenanceOwnWindow:           3,
	}
	worst := provenances[0]
	worstRank := order[worst]
	for _, p := range provenances[1:] {
		if rank, ok := order[p]; ok && rank < worstRank {
			worst = p
			worstRank = rank
		}
	}
	return worst
}

func SortNodeIDs(ids []string) []string {
	out := append([]string(nil), ids...)
	sort.Strings(out)
	return out
}
