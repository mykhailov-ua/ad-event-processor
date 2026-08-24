package fraud

import (
	"context"
	"encoding/hex"
	"log/slog"
	"sync"
	"time"

	"ad-event-processor/internal/database"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const campaignConfigCacheTTL = 60 * time.Second

type campaignFraudConfig struct {
	pass    uint8
	suspect uint8
	ivt     uint8
	block   uint8
	ghost   bool
}

type fraudScoringRule struct {
	q         *database.CHQuery
	writeConn driver.Conn
	pool      *pgxpool.Pool
	scorer    Scorer
	batchSize int

	campaignMu      sync.Mutex
	campaignConfigs map[string]campaignFraudConfig
	campaignExpiry  time.Time
}

func NewFraudScoringRule(q *database.CHQuery, writeConn driver.Conn, pool *pgxpool.Pool, scorer Scorer, batchSize int) Rule {
	return &fraudScoringRule{
		q:         q,
		writeConn: writeConn,
		pool:      pool,
		scorer:    scorer,
		batchSize: batchSize,
	}
}

func (r *fraudScoringRule) fetchCampaignConfigs(ctx context.Context) (map[string]campaignFraudConfig, error) {
	if r.pool == nil {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, "SELECT id, fraud_threshold_pass, fraud_threshold_suspect, fraud_threshold_ivt, fraud_threshold_block, ghost_ivt_enabled FROM campaigns")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := make(map[string]campaignFraudConfig)
	for rows.Next() {
		var id uuid.UUID
		var pass, suspect, ivt, block int16
		var ghost bool
		if err := rows.Scan(&id, &pass, &suspect, &ivt, &block, &ghost); err != nil {
			return nil, err
		}
		configs[id.String()] = campaignFraudConfig{
			pass:    uint8(pass),
			suspect: uint8(suspect),
			ivt:     uint8(ivt),
			block:   uint8(block),
			ghost:   ghost,
		}
	}
	return configs, nil
}

func (r *fraudScoringRule) getCampaignConfigs(ctx context.Context) (map[string]campaignFraudConfig, error) {
	r.campaignMu.Lock()
	defer r.campaignMu.Unlock()

	now := time.Now()
	if r.campaignConfigs != nil && now.Before(r.campaignExpiry) {
		return r.campaignConfigs, nil
	}

	configs, err := r.fetchCampaignConfigs(ctx)
	if err != nil {
		return nil, err
	}
	r.campaignConfigs = configs
	r.campaignExpiry = now.Add(campaignConfigCacheTTL)
	return configs, nil
}

func (r *fraudScoringRule) Name() string {
	return "fraud_scoring_shadow"
}

func (r *fraudScoringRule) Find(ctx context.Context) ([]SuspiciousIP, error) {
	if r.q == nil || r.scorer == nil {
		return nil, nil
	}

	configs, err := r.getCampaignConfigs(ctx)
	if err != nil {
		slog.Warn("fraud shadow scoring: failed to fetch campaign configs from postgres", "error", err)
	}

	query := `
SELECT
    window_start,
    ip_hash,
    campaign_id,
    events,
    clicks,
    spend_micro,
    budget_limit_micro,
    unique_users,
    unique_uas
FROM ml_features_1m
WHERE window_start >= subtractMinutes(now(), ?)
ORDER BY window_start DESC
LIMIT ?`

	rows, err := r.q.Query(ctx, query, 5, r.batchSize)
	if err != nil {
		fraudScoringErrorsTotal.Inc()
		slog.Warn("fraud shadow scoring skipped: clickhouse query failed", "error", err)
		return nil, nil
	}
	defer func() { _ = rows.Close() }()

	var featureRows []FeatureRow
	for rows.Next() {
		var fr FeatureRow
		var campaignID string
		var ipHash []byte
		if err := rows.Scan(
			&fr.WindowStart,
			&ipHash,
			&campaignID,
			&fr.Events,
			&fr.Clicks,
			&fr.SpendMicro,
			&fr.BudgetLimitMicro,
			&fr.UniqueUsers,
			&fr.UniqueUAs,
		); err != nil {
			fraudScoringErrorsTotal.Inc()
			slog.Warn("fraud shadow scoring skipped: clickhouse scan failed", "error", err)
			return nil, nil
		}
		fr.CampaignID = campaignID
		fr.IPAddress = hex.EncodeToString(ipHash)
		featureRows = append(featureRows, fr)
	}

	if len(featureRows) == 0 {
		return nil, nil
	}

	fraudScoringCandidatesTotal.Add(float64(len(featureRows)))

	start := time.Now()
	scores, err := r.scorer.ScoreBatch(ctx, featureRows)
	duration := time.Since(start).Seconds()
	fraudScoringDurationSeconds.Observe(duration)

	if err != nil {
		fraudScoringErrorsTotal.Inc()
		slog.Warn("fraud shadow scoring skipped: model inference failed", "error", err)
		return nil, nil
	}

	if r.writeConn != nil {
		if err := r.insertShadowScores(ctx, featureRows, scores); err != nil {
			slog.Error("failed to insert ml shadow scores batch to clickhouse", "error", err, "rows", len(scores))
		}
	}

	var out []SuspiciousIP
	for i, score := range scores {
		ip := featureRows[i].IPAddress

		pass := uint8(0)
		suspect := uint8(0)
		ivt := uint8(0)
		block := uint8(0)
		ghostEnabled := false

		if configs != nil {
			if cfg, ok := configs[featureRows[i].CampaignID]; ok {
				pass = cfg.pass
				suspect = cfg.suspect
				ivt = cfg.ivt
				block = cfg.block
				ghostEnabled = cfg.ghost
			}
		}

		decision := DecideWithCampaign(featureRows[i], score, pass, suspect, ivt, block)
		fraudScore := decision.Score
		var action string

		switch decision.Tier {
		case FraudTierSuspect:
			action = "boost"
			out = append(out, SuspiciousIP{
				IP:         ip,
				Reason:     r.scorer.Name(),
				Score:      float64(fraudScore),
				CampaignID: featureRows[i].CampaignID,
				Action:     "boost",
				Boost:      int32(fraudScore),
				TTLSeconds: 300,
			})
		case FraudTierIVT:
			if ghostEnabled {
				action = "ghost"
				out = append(out, SuspiciousIP{
					IP:         ip,
					Reason:     r.scorer.Name(),
					Score:      float64(fraudScore),
					CampaignID: featureRows[i].CampaignID,
					Action:     "ghost",
					Boost:      int32(fraudScore),
					TTLSeconds: 300,
				})
			}
		case FraudTierBlock:
			action = "blacklist"
			out = append(out, SuspiciousIP{
				IP:         ip,
				Reason:     r.scorer.Name(),
				Score:      float64(fraudScore),
				CampaignID: featureRows[i].CampaignID,
				Action:     "blacklist",
				Boost:      int32(fraudScore),
				TTLSeconds: 3600,
			})
		default:
			slog.Debug("ml shadow score",
				"ip", ip,
				"fraud_shadow_score", score,
				"tier", decision.Tier,
				"model", r.scorer.Name(),
			)
		}
		recordShadowMetrics(score, decision.Tier, action)
	}

	slog.Info("ml shadow batch",
		"rows", len(featureRows),
		"actions", len(out),
		"model", r.scorer.Name(),
		"duration_sec", duration,
	)

	return out, nil
}
