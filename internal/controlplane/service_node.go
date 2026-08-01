package controlplane

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	db "espx/internal/domain/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const upsertNodeCapacityScoreSQL = `
INSERT INTO node_capacity_scores (
    node_id, region_code, role, score, weight, provenance, epoch_id, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (node_id, region_code, role) DO UPDATE SET
    score = EXCLUDED.score,
    weight = EXCLUDED.weight,
    provenance = EXCLUDED.provenance,
    epoch_id = EXCLUDED.epoch_id,
    updated_at = NOW()
`

const upsertRegionTrafficDialSQL = `
INSERT INTO region_traffic_dial (
    region_code, score, weight, provenance, epoch_id, updated_at
) VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (region_code) DO UPDATE SET
    score = EXCLUDED.score,
    weight = EXCLUDED.weight,
    provenance = EXCLUDED.provenance,
    epoch_id = EXCLUDED.epoch_id,
    updated_at = NOW()
`

var regionalScorerRoles = []string{RoleTracker, RoleRegionProxy, RoleProcessor}

type NodeCapacityScorer struct {
	svc    *Service
	pool   *pgxpool.Pool
	region int16
	cfg    ScorerConfig
	epoch  atomic.Int64
	states sync.Map
}

func NewNodeCapacityScorer(svc *Service) *NodeCapacityScorer {
	region := int16(0)
	cfg := DefaultScorerConfig()
	if svc != nil && svc.cfg != nil {
		region = int16(svc.cfg.RegionCode)
		cfg = ScorerConfigFrom(svc.cfg)
	}
	return &NodeCapacityScorer{
		svc:    svc,
		pool:   svc.GetPool(),
		region: region,
		cfg:    cfg,
	}
}

func (s *NodeCapacityScorer) Tick(ctx context.Context, now time.Time) error {
	if s == nil || s.pool == nil {
		return nil
	}
	run := func(runCtx context.Context) error {
		epoch := s.epoch.Add(1)
		for _, role := range regionalScorerRoles {
			if err := s.scoreRole(runCtx, role, now, epoch); err != nil {
				return fmt.Errorf("score role=%s region=%d: %w", role, s.region, err)
			}
		}
		return nil
	}
	if s.svc != nil {
		return s.svc.withPgLow(ctx, run)
	}
	return run(ctx)
}

func (s *NodeCapacityScorer) scoreRole(ctx context.Context, role string, now time.Time, epoch int64) error {
	window := time.Duration(s.cfg.WindowMin) * time.Minute
	if window <= 0 {
		window = time.Duration(defaultScoreWindowMin) * time.Minute
	}
	windowStart := now.Add(-window)
	windowEnd := now

	q := db.New(s.pool)
	rows, err := q.ListNodeMetricBucketsRegionRoleWindow(ctx, db.ListNodeMetricBucketsRegionRoleWindowParams{
		RegionCode: s.region,
		Role:       role,
		BucketTs:   pgtype.Timestamptz{Time: windowStart, Valid: true},
		BucketTs_2: pgtype.Timestamptz{Time: windowEnd, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("list metric buckets role=%s: %w", role, err)
	}
	if len(rows) == 0 {
		return nil
	}

	nodeBuckets := groupBucketsByNode(rows)
	prevWeights := s.loadPreviousWeights(ctx, q, role)
	defs := MetricsForRole(role)
	if s.svc != nil && s.svc.scoringWeights != nil {
		defs = s.svc.scoringWeights.MetricsForRole(role)
	}
	historical := s.loadHistoricalByMetric(ctx, q, role, defs, now)
	neighbors := buildNeighborMediansByNode(nodeBuckets, defs)

	nodeIDs := SortNodeIDs(keys(nodeBuckets))
	rawWeights := make([]float64, 0, len(nodeIDs))
	results := make([]scoredNode, 0, len(nodeIDs))

	for _, nodeID := range nodeIDs {
		buckets := nodeBuckets[nodeID]
		capacity, provenance, diskDeg, budgetFail := scoreNodeFromBuckets(
			nodeID, role, buckets, defs, neighbors[nodeID], historical, now, windowStart, s.cfg,
		)

		prevWeight := prevWeights[nodeID]
		if prevWeight <= 0 {
			prevWeight = 1.0 / float64(maxInt(1, len(nodeIDs)))
		}
		stateKey := nodeID + "\x00" + role
		var state NodeScoreState
		if v, ok := s.states.Load(stateKey); ok {
			state = v.(NodeScoreState)
		}
		weightRes := ScoreNode(NodeScoreInput{
			Uptime:         window + time.Minute,
			Kind:           MetricUtilization,
			Buckets:        []BucketPoint{{Mean: capacity, SampleCount: int64(s.cfg.MinSamples)}},
			PreviousWeight: prevWeight,
			State:          state,
		}, s.cfg)
		weight := ApplyHardSignals(weightRes.Weight, diskDeg, budgetFail)
		if diskDeg || budgetFail {
			provenance = ProvenanceConservativeDefault
		}
		s.states.Store(stateKey, NodeScoreState{EMAScore: weightRes.Score, DrainEpochs: weightRes.DrainEpochs})
		rawWeights = append(rawWeights, weight)
		results = append(results, scoredNode{
			nodeID:     nodeID,
			score:      capacity,
			weight:     weight,
			provenance: provenance,
			diskDeg:    diskDeg,
			budgetFail: budgetFail,
			smoothed:   weightRes.Score,
		})
	}

	normWeights := NormalizePeerWeights(rawWeights, s.cfg.WeightMin, s.cfg.WeightMax)

	for i := range results {
		if results[i].diskDeg || results[i].budgetFail {
			normWeights[i] = 0
		}
	}
	normWeights = NormalizePeerWeights(normWeights, s.cfg.WeightMin, s.cfg.WeightMax)

	batch := &pgx.Batch{}
	for i, res := range results {
		batch.Queue(
			upsertNodeCapacityScoreSQL,
			res.nodeID,
			s.region,
			role,
			res.score,
			normWeights[i],
			res.provenance,
			epoch,
		)
	}
	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("upsert node capacity score batch item %d: %w", i, err)
		}
	}
	return nil
}

type scoredNode struct {
	nodeID     string
	score      float64
	weight     float64
	provenance string
	diskDeg    bool
	budgetFail bool
	smoothed   float64
}

func scoreNodeFromBuckets(
	nodeID, role string,
	buckets []db.NodeMetricBucket,
	defs []ScoringMetricDef,
	neighborByMetric map[string][]float64,
	historical map[string]*float64,
	now time.Time,
	windowStart time.Time,
	cfg ScorerConfig,
) (capacity float64, provenance string, diskDegraded, budgetInvariantFail bool) {
	_ = nodeID
	metricBuckets := groupBucketsByMetric(buckets)
	values := make(map[string]float64, len(metricBuckets))
	for metric, pts := range metricBuckets {
		values[metric] = latestMean(pts)
	}
	diskDegraded, budgetInvariantFail = HardSignalsActive(values)

	var provenances []string
	laneValues := make(map[string]float64, len(defs))
	uptime := nodeUptime(buckets, now, cfg.WindowMin)

	for _, def := range defs {
		pts := metricBuckets[def.Name]
		bucketPts := toBucketPoints(pts)
		lane := ScoreNode(NodeScoreInput{
			Uptime:           uptime,
			Kind:             def.Kind,
			Buckets:          bucketPts,
			NeighborValues:   neighborByMetric[def.Name],
			HistoricalValue:  historical[def.Name],
			PreviousWeight:   1,
			ScrapeMissEpochs: scrapeMissEpochs(pts, windowStart),
		}, cfg)
		laneValues[def.Name] = lane.RawValue
		provenances = append(provenances, lane.Provenance)
	}

	capacity = ComputeCapacityScoreFromValues(role, laneValues, defs)
	return capacity, DominantProvenance(provenances), diskDegraded, budgetInvariantFail
}

func (s *NodeCapacityScorer) loadHistoricalByMetric(
	ctx context.Context,
	q *db.Queries,
	role string,
	defs []ScoringMetricDef,
	now time.Time,
) map[string]*float64 {
	kindByMetric := make(map[string]MetricKind, len(defs))
	for _, def := range defs {
		kindByMetric[def.Name] = def.Kind
	}
	day := HistoricalSnapshotDay(now)
	rows, err := q.ListNodeMetricDailySnapshotsByDay(ctx, db.ListNodeMetricDailySnapshotsByDayParams{
		Day:        pgtype.Date{Time: day, Valid: true},
		RegionCode: s.region,
		Role:       role,
	})
	if err != nil {
		return nil
	}
	out := make(map[string]*float64, len(rows))
	for _, row := range rows {
		kind, ok := kindByMetric[row.Metric]
		if !ok {
			continue
		}
		raw, ok := historicalRawFromSnapshot(row, kind)
		if !ok {
			continue
		}
		v := raw
		out[row.Metric] = &v
	}
	return out
}

func buildNeighborMediansByNode(
	nodeBuckets map[string][]db.NodeMetricBucket,
	defs []ScoringMetricDef,
) map[string]map[string][]float64 {
	nodeRaw := make(map[string]map[string]float64, len(nodeBuckets))
	for nodeID, buckets := range nodeBuckets {
		byMetric := groupBucketsByMetric(buckets)
		nodeRaw[nodeID] = make(map[string]float64, len(defs))
		for _, def := range defs {
			raw, ok := aggregateWindow(toBucketPoints(byMetric[def.Name]), def.Kind)
			if ok {
				nodeRaw[nodeID][def.Name] = raw
			}
		}
	}
	type metricPeer struct {
		nodeID string
		value  float64
	}
	metricPeers := make(map[string][]metricPeer, len(defs))
	for nodeID, metrics := range nodeRaw {
		for name, v := range metrics {
			metricPeers[name] = append(metricPeers[name], metricPeer{nodeID: nodeID, value: v})
		}
	}

	out := make(map[string]map[string][]float64, len(nodeBuckets))
	for nodeID := range nodeBuckets {
		lanes := make(map[string][]float64, len(defs))
		for _, def := range defs {
			peers := metricPeers[def.Name]
			if len(peers) <= 1 {
				continue
			}
			lane := make([]float64, 0, len(peers)-1)
			for _, peer := range peers {
				if peer.nodeID != nodeID {
					lane = append(lane, peer.value)
				}
			}
			if len(lane) > 0 {
				lanes[def.Name] = lane
			}
		}
		out[nodeID] = lanes
	}
	return out
}

func (s *NodeCapacityScorer) loadPreviousWeights(ctx context.Context, q *db.Queries, role string) map[string]float64 {
	rows, err := q.ListNodeCapacityScoresByRegionRole(ctx, db.ListNodeCapacityScoresByRegionRoleParams{
		RegionCode: s.region,
		Role:       role,
	})
	if err != nil {
		return nil
	}
	out := make(map[string]float64, len(rows))
	for _, row := range rows {
		out[row.NodeID] = row.Weight
	}
	return out
}

func groupBucketsByNode(rows []db.NodeMetricBucket) map[string][]db.NodeMetricBucket {
	out := make(map[string][]db.NodeMetricBucket)
	for _, row := range rows {
		out[row.NodeID] = append(out[row.NodeID], row)
	}
	return out
}

func groupBucketsByMetric(rows []db.NodeMetricBucket) map[string][]db.NodeMetricBucket {
	out := make(map[string][]db.NodeMetricBucket)
	for _, row := range rows {
		out[row.Metric] = append(out[row.Metric], row)
	}
	return out
}

func toBucketPoints(rows []db.NodeMetricBucket) []BucketPoint {
	pts := make([]BucketPoint, 0, len(rows))
	for _, row := range rows {
		pts = append(pts, BucketPoint{
			P50:         row.ValueP50.Float64,
			P99:         row.ValueP99.Float64,
			Mean:        row.ValueMean.Float64,
			SampleCount: row.SampleCount,
		})
	}
	return pts
}

func latestMean(rows []db.NodeMetricBucket) float64 {
	if len(rows) == 0 {
		return 0
	}
	last := rows[len(rows)-1]
	if last.ValueMean.Valid {
		return last.ValueMean.Float64
	}
	return 0
}

func nodeUptime(rows []db.NodeMetricBucket, now time.Time, windowMin int) time.Duration {
	if len(rows) == 0 {
		return 0
	}
	oldest := rows[0].BucketTs.Time
	for _, row := range rows[1:] {
		if row.BucketTs.Time.Before(oldest) {
			oldest = row.BucketTs.Time
		}
	}
	uptime := now.Sub(oldest)
	warmup := time.Duration(windowMin) * time.Minute
	if uptime > warmup {
		return uptime
	}
	return uptime
}

func scrapeMissEpochs(rows []db.NodeMetricBucket, windowStart time.Time) int {
	if len(rows) == 0 {
		return maxScrapeMissEpochs + 1
	}
	latest := rows[len(rows)-1].BucketTs.Time
	if latest.Before(windowStart.Add(30 * time.Second)) {
		return maxScrapeMissEpochs + 1
	}
	return 0
}

func keys(m map[string][]db.NodeMetricBucket) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

type GlobalRegionTrafficScorer struct {
	svc    *Service
	pool   *pgxpool.Pool
	cfg    ScorerConfig
	epoch  atomic.Int64
	states sync.Map
}

func NewGlobalRegionTrafficScorer(svc *Service) *GlobalRegionTrafficScorer {
	cfg := DefaultScorerConfig()
	if svc != nil && svc.cfg != nil {
		cfg = ScorerConfigFrom(svc.cfg)
	}
	return &GlobalRegionTrafficScorer{
		svc:  svc,
		pool: svc.GetPool(),
		cfg:  cfg,
	}
}

func (g *GlobalRegionTrafficScorer) Tick(ctx context.Context, now time.Time) error {
	if g == nil || g.pool == nil {
		return nil
	}
	if g.svc == nil || g.svc.cfg == nil || !g.svc.cfg.MultiRegionGlobal() {
		return nil
	}
	run := func(runCtx context.Context) error {
		return g.tick(runCtx, now, g.epoch.Add(1))
	}
	if g.svc != nil {
		return g.svc.withPgLow(ctx, run)
	}
	return run(ctx)
}

func (g *GlobalRegionTrafficScorer) tick(ctx context.Context, now time.Time, epoch int64) error {
	q := db.New(g.pool)
	regions, err := q.ListActiveRegionCodes(ctx)
	if err != nil {
		return fmt.Errorf("list active region codes: %w", err)
	}
	if len(regions) == 0 {
		return nil
	}

	trackerScores, err := q.ListNodeCapacityScoresByRole(ctx, RoleTracker)
	if err != nil {
		return fmt.Errorf("list node capacity scores role=%s: %w", RoleTracker, err)
	}
	nodesByRegion := make(map[int16][]db.NodeCapacityScore, len(regions))
	for _, row := range trackerScores {
		nodesByRegion[row.RegionCode] = append(nodesByRegion[row.RegionCode], row)
	}

	prevWeights := g.loadPreviousDialWeights(ctx, q)
	inputs := make([]RegionDialInput, 0, len(regions))
	for _, region := range regions {
		nodes := nodesByRegion[region]
		if len(nodes) == 0 {
			continue
		}
		var state NodeScoreState
		if v, ok := g.states.Load(region); ok {
			state = v.(NodeScoreState)
		}
		inputs = append(inputs, RegionDialInput{
			RegionCode: region,
			Nodes:      nodes,
			PrevWeight: prevWeights[region],
			State:      state,
		})
	}
	if len(inputs) == 0 {
		return nil
	}

	results := ComputeRegionDialResults(inputs, g.cfg)
	if len(results) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, res := range results {
		g.states.Store(res.RegionCode, res.State)
		batch.Queue(
			upsertRegionTrafficDialSQL,
			res.RegionCode,
			res.Score,
			res.Weight,
			res.Provenance,
			epoch,
		)
	}
	br := g.pool.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("upsert region traffic dial batch item %d: %w", i, err)
		}
	}
	return nil
}

func (g *GlobalRegionTrafficScorer) loadPreviousDialWeights(ctx context.Context, q *db.Queries) map[int16]float64 {
	rows, err := q.ListRegionTrafficDial(ctx)
	if err != nil {
		return nil
	}
	out := make(map[int16]float64, len(rows))
	for _, row := range rows {
		out[row.RegionCode] = row.Weight
	}
	return out
}
