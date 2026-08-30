package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	banditMinClicks       = 100
	AlgorithmThompson     = "thompson"
	AlgorithmProportional = "proportional"
)

type BanditFlowFilter struct {
	FlowID     *uuid.UUID
	CampaignID *uuid.UUID
}

type BanditApplyConfig struct {
	MinClicks         int64
	MinSpendMicro     int64
	Scope             string
	Algorithm         string
	Objective         string
	MaxWeightDeltaPct int
}

func defaultBanditApplyConfig(cfg BanditApplyConfig) BanditApplyConfig {
	if cfg.MinClicks <= 0 {
		cfg.MinClicks = banditMinClicks
	}
	return cfg
}

type banditPathJSON struct {
	Weight  int32              `json:"weight"`
	Landers []banditLanderJSON `json:"landers"`
	Offers  []banditOfferJSON  `json:"offers"`
}

type banditLanderJSON struct {
	LanderID uuid.UUID `json:"lander_id"`
	Weight   int32     `json:"weight"`
}

type banditOfferJSON struct {
	OfferID uuid.UUID `json:"offer_id"`
	Weight  int32     `json:"weight"`
}

func OptimizeFlowBanditTx(ctx context.Context, tx pgx.Tx, host BanditHost) ([]uuid.UUID, error) {
	if host == nil {
		return nil, nil
	}
	lookbackDays := host.MABLookbackDays()
	if lookbackDays <= 0 {
		lookbackDays = 90
	}
	lookbackEnd := time.Now().UTC()
	lookbackStart := lookbackEnd.Add(-time.Duration(lookbackDays) * 24 * time.Hour)
	return optimizeFlowBanditFilteredTx(ctx, tx, host, BanditFlowFilter{}, defaultBanditApplyConfig(BanditApplyConfig{}), lookbackStart, lookbackEnd)
}

func OptimizeFlowBanditFilteredTx(
	ctx context.Context,
	tx pgx.Tx,
	host BanditHost,
	filter BanditFlowFilter,
	cfg BanditApplyConfig,
	lookbackStart, lookbackEnd time.Time,
) ([]uuid.UUID, error) {
	return optimizeFlowBanditFilteredTx(ctx, tx, host, filter, defaultBanditApplyConfig(cfg), lookbackStart, lookbackEnd)
}

func optimizeFlowBanditFilteredTx(
	ctx context.Context,
	tx pgx.Tx,
	host BanditHost,
	filter BanditFlowFilter,
	cfg BanditApplyConfig,
	lookbackStart, lookbackEnd time.Time,
) ([]uuid.UUID, error) {
	if host == nil {
		return nil, nil
	}
	cfg = defaultBanditApplyConfig(cfg)

	landerStats, offerStats, err := host.QueryFlowBanditStats(ctx, lookbackStart, lookbackEnd)
	if err != nil {
		return nil, err
	}
	if landerStats == nil && offerStats == nil {
		return nil, nil
	}

	var flowIDParam, campaignIDParam *uuid.UUID
	if filter.FlowID != nil {
		flowIDParam = filter.FlowID
	}
	if filter.CampaignID != nil {
		campaignIDParam = filter.CampaignID
	}

	rows, err := tx.Query(ctx, `
		SELECT f.id, f.paths, c.id
		FROM flows f
		JOIN campaigns c ON c.flow_id = f.id AND c.deleted_at IS NULL
		WHERE ($1::uuid IS NULL OR f.id = $1)
		  AND ($2::uuid IS NULL OR c.id = $2)`,
		flowIDParam, campaignIDParam)
	if err != nil {
		return nil, fmt.Errorf("flow bandit list flows: %w", err)
	}
	defer rows.Close()

	type flowRow struct {
		id         uuid.UUID
		paths      []byte
		campaignID uuid.UUID
	}
	var flows []flowRow
	for rows.Next() {
		var r flowRow
		if err := rows.Scan(&r.id, &r.paths, &r.campaignID); err != nil {
			return nil, err
		}
		flows = append(flows, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(flows) == 0 {
		return nil, nil
	}

	byFlow := make(map[uuid.UUID]struct {
		paths     []byte
		campaigns []uuid.UUID
	})
	for _, r := range flows {
		entry := byFlow[r.id]
		entry.paths = r.paths
		entry.campaigns = append(entry.campaigns, r.campaignID)
		byFlow[r.id] = entry
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var publishCampaigns []uuid.UUID
	updateBatch := &pgx.Batch{}

	for flowID, entry := range byFlow {
		var newPaths []byte
		var changed bool
		var err error
		if cfg.Algorithm == AlgorithmProportional {
			newPaths, changed, err = ApplyFlowBanditProportional(entry.paths, entry.campaigns, landerStats, offerStats, cfg)
		} else {
			newPaths, changed, err = ApplyFlowBanditThompson(entry.paths, entry.campaigns, landerStats, offerStats, rng, cfg)
		}
		if err != nil {
			return nil, fmt.Errorf("flow bandit apply %s: %w", flowID, err)
		}
		if !changed {
			continue
		}
		updateBatch.Queue(`UPDATE flows SET paths = $2::jsonb, updated_at = now() WHERE id = $1`, flowID, newPaths)
		publishCampaigns = append(publishCampaigns, entry.campaigns...)
	}

	if updateBatch.Len() == 0 {
		return nil, nil
	}
	br := tx.SendBatch(ctx, updateBatch)
	defer func() { _ = br.Close() }()
	for i := range updateBatch.Len() {
		if _, err := br.Exec(); err != nil {
			return nil, fmt.Errorf("flow bandit update batch %d: %w", i, err)
		}
	}
	return publishCampaigns, nil
}

func ApplyFlowBanditThompson(
	raw []byte,
	campaignIDs []uuid.UUID,
	landerByCampaign map[uuid.UUID]map[uuid.UUID]EntityBanditStat,
	offerByCampaign map[uuid.UUID]map[uuid.UUID]EntityBanditStat,
	rng *rand.Rand,
	cfg BanditApplyConfig,
) ([]byte, bool, error) {
	cfg = defaultBanditApplyConfig(cfg)
	var paths []banditPathJSON
	if err := json.Unmarshal(raw, &paths); err != nil || len(paths) == 0 {
		return nil, false, err
	}
	aggLanders := aggregateEntityBanditStats(campaignIDs, landerByCampaign)
	aggOffers := aggregateEntityBanditStats(campaignIDs, offerByCampaign)

	applyLanders := cfg.Scope == "" || cfg.Scope == "lander"
	applyOffers := cfg.Scope == "" || cfg.Scope == "offer"

	changed := false
	for i := range paths {
		if applyLanders && applyThompsonLanders(&paths[i].Landers, aggLanders, rng, cfg) {
			changed = true
		}
		if applyOffers && applyThompsonOffers(&paths[i].Offers, aggOffers, rng, cfg) {
			changed = true
		}
	}
	if !changed {
		return nil, false, nil
	}
	out, err := json.Marshal(paths)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func ApplyFlowBanditProportional(
	raw []byte,
	campaignIDs []uuid.UUID,
	landerByCampaign map[uuid.UUID]map[uuid.UUID]EntityBanditStat,
	offerByCampaign map[uuid.UUID]map[uuid.UUID]EntityBanditStat,
	cfg BanditApplyConfig,
) ([]byte, bool, error) {
	cfg = defaultBanditApplyConfig(cfg)
	objective := cfg.Objective
	if objective == "" {
		objective = BanditObjectiveEPC
	}
	var paths []banditPathJSON
	if err := json.Unmarshal(raw, &paths); err != nil || len(paths) == 0 {
		return nil, false, err
	}
	aggLanders := aggregateEntityBanditStats(campaignIDs, landerByCampaign)
	aggOffers := aggregateEntityBanditStats(campaignIDs, offerByCampaign)

	applyLanders := cfg.Scope == "" || cfg.Scope == "lander"
	applyOffers := cfg.Scope == "" || cfg.Scope == "offer"

	changed := false
	for i := range paths {
		if applyLanders && applyProportionalLanders(&paths[i].Landers, aggLanders, objective, cfg) {
			changed = true
		}
		if applyOffers && applyProportionalOffers(&paths[i].Offers, aggOffers, objective, cfg) {
			changed = true
		}
	}
	if !changed {
		return nil, false, nil
	}
	out, err := json.Marshal(paths)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func aggregateEntityBanditStats(
	campaignIDs []uuid.UUID,
	byCampaign map[uuid.UUID]map[uuid.UUID]EntityBanditStat,
) map[uuid.UUID]EntityBanditStat {
	out := make(map[uuid.UUID]EntityBanditStat)
	for _, campID := range campaignIDs {
		perEntity, ok := byCampaign[campID]
		if !ok {
			continue
		}
		for entityID, st := range perEntity {
			cur := out[entityID]
			cur.Clicks += st.Clicks
			cur.Conversions += st.Conversions
			cur.Payout += st.Payout
			cur.SpendMicro += st.SpendMicro
			out[entityID] = cur
		}
	}
	return out
}

func applyThompsonLanders(landers *[]banditLanderJSON, stats map[uuid.UUID]EntityBanditStat, rng *rand.Rand, cfg BanditApplyConfig) bool {
	if landers == nil || len(*landers) < 2 {
		return false
	}
	arms, totalClicks := banditLanderArms(*landers, stats)
	if totalClicks < cfg.MinClicks || len(arms) < 2 {
		return false
	}
	weights := ThompsonWeights(arms, rng)
	return applyEntityWeights(landers, weights, cfg.MaxWeightDeltaPct, func(l *banditLanderJSON, w int32) { l.Weight = w })
}

func applyThompsonOffers(offers *[]banditOfferJSON, stats map[uuid.UUID]EntityBanditStat, rng *rand.Rand, cfg BanditApplyConfig) bool {
	if offers == nil || len(*offers) < 2 {
		return false
	}
	arms, totalClicks := banditOfferArms(*offers, stats)
	if totalClicks < cfg.MinClicks || len(arms) < 2 {
		return false
	}
	weights := ThompsonWeights(arms, rng)
	return applyOfferWeights(offers, weights, cfg.MaxWeightDeltaPct)
}

func applyProportionalLanders(landers *[]banditLanderJSON, stats map[uuid.UUID]EntityBanditStat, objective string, cfg BanditApplyConfig) bool {
	if landers == nil || len(*landers) < 2 {
		return false
	}
	entityIDs, totalClicks := proportionalEntityIDs(*landers, stats, func(l banditLanderJSON) uuid.UUID { return l.LanderID })
	if totalClicks < cfg.MinClicks {
		return false
	}
	weights := ProportionalWeights(objective, stats, entityIDs, cfg.MinSpendMicro)
	if len(weights) < 2 {
		return false
	}
	return applyEntityWeights(landers, weights, cfg.MaxWeightDeltaPct, func(l *banditLanderJSON, w int32) { l.Weight = w })
}

func applyProportionalOffers(offers *[]banditOfferJSON, stats map[uuid.UUID]EntityBanditStat, objective string, cfg BanditApplyConfig) bool {
	if offers == nil || len(*offers) < 2 {
		return false
	}
	entityIDs, totalClicks := proportionalEntityIDs(*offers, stats, func(o banditOfferJSON) uuid.UUID { return o.OfferID })
	if totalClicks < cfg.MinClicks {
		return false
	}
	weights := ProportionalWeights(objective, stats, entityIDs, cfg.MinSpendMicro)
	if len(weights) < 2 {
		return false
	}
	return applyOfferWeights(offers, weights, cfg.MaxWeightDeltaPct)
}

func proportionalEntityIDs[T any](items []T, stats map[uuid.UUID]EntityBanditStat, idFn func(T) uuid.UUID) ([]uuid.UUID, int64) {
	entityIDs := make([]uuid.UUID, 0, len(items))
	var totalClicks int64
	for _, item := range items {
		id := idFn(item)
		entityIDs = append(entityIDs, id)
		totalClicks += stats[id].Clicks
	}
	return entityIDs, totalClicks
}

func banditLanderArms(
	landers []banditLanderJSON,
	stats map[uuid.UUID]EntityBanditStat,
) (map[uuid.UUID]ArmStat, int64) {
	arms := make(map[uuid.UUID]ArmStat)
	var totalClicks int64
	for _, l := range landers {
		st := stats[l.LanderID]
		totalClicks += st.Clicks
		if st.Clicks > 0 {
			arms[l.LanderID] = ArmStat{Clicks: st.Clicks, Conversions: st.Conversions}
		}
	}
	return arms, totalClicks
}

func banditOfferArms(offers []banditOfferJSON, stats map[uuid.UUID]EntityBanditStat) (map[uuid.UUID]ArmStat, int64) {
	arms := make(map[uuid.UUID]ArmStat)
	var totalClicks int64
	for _, o := range offers {
		st := stats[o.OfferID]
		totalClicks += st.Clicks
		if st.Clicks > 0 {
			arms[o.OfferID] = ArmStat{Clicks: st.Clicks, Conversions: st.Conversions}
		}
	}
	return arms, totalClicks
}

func applyEntityWeights(landers *[]banditLanderJSON, weights map[uuid.UUID]int32, maxDeltaPct int, setWeight func(*banditLanderJSON, int32)) bool {
	changed := false
	for i := range *landers {
		w, ok := weights[(*landers)[i].LanderID]
		if !ok || w <= 0 {
			continue
		}
		current := (*landers)[i].Weight
		if current <= 0 {
			current = 1
		}
		clamped := clampProposedWeight(current, w, maxDeltaPct)
		if (*landers)[i].Weight != clamped {
			setWeight(&(*landers)[i], clamped)
			changed = true
		}
	}
	return changed
}

func applyOfferWeights(offers *[]banditOfferJSON, weights map[uuid.UUID]int32, maxDeltaPct int) bool {
	changed := false
	for i := range *offers {
		w, ok := weights[(*offers)[i].OfferID]
		if !ok || w <= 0 {
			continue
		}
		current := (*offers)[i].Weight
		if current <= 0 {
			current = 1
		}
		clamped := clampProposedWeight(current, w, maxDeltaPct)
		if (*offers)[i].Weight != clamped {
			(*offers)[i].Weight = clamped
			changed = true
		}
	}
	return changed
}
