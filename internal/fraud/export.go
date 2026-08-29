package fraud

import (
	"ad-event-processor/internal/config"
	adminhooks "ad-event-processor/internal/fraud/admin_hooks"
	"ad-event-processor/internal/fraud/features"
	"ad-event-processor/internal/fraud/scorer"
	"ad-event-processor/pkg/piihash"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	redis "github.com/redis/go-redis/v9"
)

type (
	Scorer                   = scorer.Scorer
	FeatureRow               = features.FeatureRow
	MicroBatcher             = scorer.MicroBatcher
	MicroBatcherConfig       = scorer.MicroBatcherConfig
	BlacklistBlocker         = adminhooks.BlacklistBlocker
	ResidentialIntelEnricher = features.ResidentialIntelEnricher
	ResidentialIntelResult   = features.ResidentialIntelResult
	FraudTier                = scorer.FraudTier
)

const (
	FraudTierPass       = scorer.FraudTierPass
	FraudTierSuspect    = scorer.FraudTierSuspect
	FraudTierIVT        = scorer.FraudTierIVT
	FraudTierBlock      = scorer.FraudTierBlock
	FraudTierPassMax    = scorer.FraudTierPassMax
	FraudTierSuspectMax = scorer.FraudTierSuspectMax
	FraudTierIVTMax     = scorer.FraudTierIVTMax
)

var FeatureNames = features.FeatureNames

func Dims() int {
	return features.Dims()
}

func NewLGBMScorer(modelPath string) (Scorer, error) {
	return scorer.NewLGBMScorer(modelPath)
}

func NewMicroBatcher(redisShards []redis.UniversalClient, s Scorer, campaignUpdateChannel string, cfg MicroBatcherConfig) *MicroBatcher {
	m := scorer.NewMicroBatcher(redisShards, s, campaignUpdateChannel, cfg)
	m.BoostScore = microbatchBoostScore
	return m
}

func DefaultMicroBatcherConfig() MicroBatcherConfig {
	return scorer.DefaultMicroBatcherConfig()
}

func SetPIIHasher(h *piihash.Hasher) {
	features.SetPIIHasher(h)
}

func ResolveManagementBlockerFromConfig(managementURL, managementPort, apiKey string) (BlacklistBlocker, error) {
	return adminhooks.ResolveManagementBlockerFromConfig(managementURL, managementPort, apiKey)
}

func NewResidentialIntelEnricherFromConfig(cfg *config.Config, redisClient redis.Cmdable, clickhouseWriteConn driver.Conn) (*ResidentialIntelEnricher, error) {
	return features.NewResidentialIntelEnricherFromConfig(cfg, redisClient, clickhouseWriteConn)
}

func MapFraudScoreTier(score int) (FraudTier, int) {
	return scorer.MapFraudScoreTier(score)
}

func MapProbabilityTier(probability float64) (FraudTier, int) {
	return scorer.MapProbabilityTier(probability)
}
