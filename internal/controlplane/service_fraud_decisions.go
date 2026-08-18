package controlplane

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/fraud"
	"github.com/bidshard/ad-event-processor/pkg/piihash"

	"github.com/google/uuid"
)

const fraudExplainCHQueryTimeout = 5 * time.Second

const fraudExplainDefaultHours = 24

type campaignFraudThresholds struct {
	pass    uint8
	suspect uint8
	ivt     uint8
	block   uint8
}

type fraudExplainCHRow struct {
	windowStart      time.Time
	campaignID       string
	events           uint64
	clicks           uint64
	spendMicro       int64
	budgetLimitMicro int64
	uniqueUsers      uint64
	uniqueUAs        uint64
	modelScore       float64
	modelName        string
	scoreCreatedAt   time.Time
	hasShadowScore   bool
}

func normalizeFraudExplainHours(hours int) int {
	if hours <= 0 {
		return fraudExplainDefaultHours
	}
	if hours > fraudExplainMaxHours {
		return fraudExplainMaxHours
	}
	return hours
}

func fraudFeatureMap(row fraud.FeatureRow) map[string]float64 {
	vec := row.ToVector()
	out := make(map[string]float64, len(fraud.FeatureNames))
	for i, name := range fraud.FeatureNames {
		out[name] = vec[i]
	}
	return out
}

func (s *Service) getCampaignFraudForCustomer(ctx context.Context, customerID, campaignID uuid.UUID) (campaignFraudThresholds, error) {
	var out campaignFraudThresholds
	if s == nil || s.GetPool() == nil {
		return out, fmt.Errorf("postgres pool not configured")
	}
	var pass, suspect, ivt, block int16
	err := s.GetPool().QueryRow(ctx, `
		SELECT fraud_threshold_pass, fraud_threshold_suspect, fraud_threshold_ivt, fraud_threshold_block
		FROM campaigns
		WHERE id = $1 AND customer_id = $2`,
		domain.ToUUID(campaignID), domain.ToUUID(customerID),
	).Scan(&pass, &suspect, &ivt, &block)
	if err != nil {
		return out, mapNotFound(err, ErrCampaignNotFound)
	}
	out.pass = uint8(pass)
	out.suspect = uint8(suspect)
	out.ivt = uint8(ivt)
	out.block = uint8(block)
	return out, nil
}

func (s *Service) queryFraudExplainCH(ctx context.Context, ipHash string, hours int, campaignID uuid.UUID) (fraudExplainCHRow, bool, error) {
	var out fraudExplainCHRow
	if s == nil || s.chQuery == nil {
		return out, false, fmt.Errorf("clickhouse not configured")
	}

	ipBytes, err := hex.DecodeString(ipHash)
	if err != nil || len(ipBytes) != 16 {
		return out, false, errValidation("ip_hash must be 32 hex characters")
	}

	query := `
SELECT
    f.window_start,
    toString(f.campaign_id) AS campaign_id,
    f.events,
    f.clicks,
    f.spend_micro,
    f.budget_limit_micro,
    f.unique_users,
    f.unique_uas,
    s.score,
    s.model_name,
    s.created_at,
    s.has_shadow_score
FROM ml_features_1m AS f
LEFT JOIN (
    SELECT
        ip_hash,
        argMax(score, created_at) AS score,
        argMax(model_name, created_at) AS model_name,
        max(created_at) AS created_at,
        count() > 0 AS has_shadow_score
    FROM ml_shadow_scores
    WHERE ip_hash = ?
      AND created_at >= subtractHours(now(), ?)
    GROUP BY ip_hash
) AS s ON f.ip_hash = s.ip_hash
WHERE f.ip_hash = ?
  AND f.window_start >= subtractHours(now(), ?)`

	args := []any{piihash.FixedString16([16]byte(ipBytes)), hours, piihash.FixedString16([16]byte(ipBytes)), hours}
	if campaignID != uuid.Nil {
		query += `
  AND f.campaign_id = ?`
		args = append(args, campaignID)
	}
	query += `
ORDER BY f.window_start DESC
LIMIT 1`

	rows, err := s.chQuery.Query(ctx, query, args...)
	if err != nil {
		return out, false, fmt.Errorf("fraud explain clickhouse query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return out, false, nil
	}

	if err := rows.Scan(
		&out.windowStart,
		&out.campaignID,
		&out.events,
		&out.clicks,
		&out.spendMicro,
		&out.budgetLimitMicro,
		&out.uniqueUsers,
		&out.uniqueUAs,
		&out.modelScore,
		&out.modelName,
		&out.scoreCreatedAt,
		&out.hasShadowScore,
	); err != nil {
		return out, false, fmt.Errorf("fraud explain clickhouse scan: %w", err)
	}
	return out, true, rows.Err()
}

func (s *Service) explainLiveScore(ctx context.Context, row fraud.FeatureRow) (float64, error) {
	scorer, err := s.explainScorer()
	if err != nil {
		return 0, err
	}
	scores, err := scorer.ScoreBatch(ctx, []fraud.FeatureRow{row})
	if err != nil {
		return 0, err
	}
	if len(scores) == 0 {
		return 0, fmt.Errorf("live scorer returned no scores")
	}
	return scores[0], nil
}

func (s *Service) explainScorer() (fraud.Scorer, error) {
	if s == nil || s.cfg == nil || !s.cfg.FraudScoring.ExplainLiveScore {
		return nil, errors.New("live fraud explain scoring disabled")
	}
	s.explainScorerMu.Lock()
	defer s.explainScorerMu.Unlock()
	if s.explainScorerInst != nil {
		return s.explainScorerInst, nil
	}
	if s.explainScorerErr != nil {
		return nil, s.explainScorerErr
	}
	modelPath := strings.TrimSpace(s.cfg.FraudScoring.ModelPath)
	if modelPath == "" {
		s.explainScorerErr = errors.New("fraud model path not configured")
		return nil, s.explainScorerErr
	}
	scorer, err := fraud.NewLGBMScorer(modelPath)
	if err != nil {
		s.explainScorerErr = err
		return nil, err
	}
	s.explainScorerInst = scorer
	return scorer, nil
}

func (s *Service) ExplainFraudDecision(ctx context.Context, customerID uuid.UUID, ipHash string, campaignID *uuid.UUID, hours int) (FraudDecisionDTO, error) {
	if customerID == uuid.Nil {
		return FraudDecisionDTO{}, errValidation("customer_id is required")
	}
	ipHash = strings.ToLower(strings.TrimSpace(ipHash))
	if err := validateMLIPHash(ipHash); err != nil {
		return FraudDecisionDTO{}, err
	}
	hours = normalizeFraudExplainHours(hours)

	var filterCampaign uuid.UUID
	if campaignID != nil && *campaignID != uuid.Nil {
		filterCampaign = *campaignID
		if _, err := s.getCampaignFraudForCustomer(ctx, customerID, filterCampaign); err != nil {
			if errors.Is(err, ErrCampaignNotFound) {
				return FraudDecisionDTO{}, ErrFraudDecisionNotFound
			}
			return FraudDecisionDTO{}, err
		}
	}

	chCtx, cancel := context.WithTimeout(ctx, fraudExplainCHQueryTimeout)
	defer cancel()

	chRow, found, err := s.queryFraudExplainCH(chCtx, ipHash, hours, filterCampaign)
	if err != nil {
		return FraudDecisionDTO{}, err
	}
	if !found {
		return FraudDecisionDTO{}, ErrFraudDecisionNotFound
	}

	resolvedCampaignID, err := uuid.Parse(strings.TrimSpace(chRow.campaignID))
	if err != nil {
		return FraudDecisionDTO{}, ErrFraudDecisionNotFound
	}

	thresholds, err := s.getCampaignFraudForCustomer(ctx, customerID, resolvedCampaignID)
	if err != nil {
		if errors.Is(err, ErrCampaignNotFound) {
			return FraudDecisionDTO{}, ErrFraudDecisionNotFound
		}
		return FraudDecisionDTO{}, err
	}

	featureRow := fraud.FeatureRow{
		WindowStart:      chRow.windowStart,
		IPAddress:        ipHash,
		CampaignID:       chRow.campaignID,
		Events:           chRow.events,
		Clicks:           chRow.clicks,
		SpendMicro:       chRow.spendMicro,
		BudgetLimitMicro: chRow.budgetLimitMicro,
		UniqueUsers:      chRow.uniqueUsers,
		UniqueUAs:        chRow.uniqueUAs,
	}

	mlProbability := 0.0
	scoreMissing := !chRow.hasShadowScore
	var modelScorePtr *float64
	modelName := strings.TrimSpace(chRow.modelName)

	if chRow.hasShadowScore {
		mlProbability = chRow.modelScore
		score := chRow.modelScore
		modelScorePtr = &score
	} else if s.cfg != nil && s.cfg.FraudScoring.ExplainLiveScore {
		liveScore, liveErr := s.explainLiveScore(chCtx, featureRow)
		if liveErr == nil {
			mlProbability = liveScore
			score := liveScore
			modelScorePtr = &score
			scoreMissing = false
			if modelName == "" {
				modelName = "live"
			}
		}
	}

	decision := fraud.DecideWithCampaign(featureRow, mlProbability, thresholds.pass, thresholds.suspect, thresholds.ivt, thresholds.block)

	evaluatedAt := chRow.windowStart.UTC()
	if chRow.hasShadowScore && !chRow.scoreCreatedAt.IsZero() {
		evaluatedAt = chRow.scoreCreatedAt.UTC()
	}

	return FraudDecisionDTO{
		IPHash:              ipHash,
		CampaignID:          resolvedCampaignID.String(),
		WindowStart:         chRow.windowStart.UTC().Format(time.RFC3339),
		EvaluatedAt:         evaluatedAt.Format(time.RFC3339),
		Disclaimer:          FraudDecisionDisclaimer(),
		Tier:                string(decision.Tier),
		Score:               decision.Score,
		MLProbability:       decision.MLProbability,
		AdjustedProbability: decision.AdjustedProbability,
		ResidentialProxy:    decision.ResidentialProxy,
		StructuralFraud:     decision.StructuralFraud,
		FPGuardApplied:      decision.FPGuardApplied,
		ModelScore:          modelScorePtr,
		ModelName:           modelName,
		ScoreMissing:        scoreMissing,
		Features:            fraudFeatureMap(featureRow),
		CampaignThresholds: FraudTierThresholdsDTO{
			Scope:      "campaign",
			PassMax:    int(thresholds.pass),
			SuspectMax: int(thresholds.suspect),
			IVTMax:     int(thresholds.ivt),
			BlockAbove: int(thresholds.block),
		},
	}, nil
}
