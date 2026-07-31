package fraud

import (
	"encoding/json"
	"os"
	"strconv"

	"espx/internal/domain"
)

type PolicyConfig struct {
	TierPass    uint8 `json:"tier_pass"`
	TierSuspect uint8 `json:"tier_suspect"`
	TierIVT     uint8 `json:"tier_ivt"`
	TierBlock   uint8 `json:"tier_block"`

	MLThreshold           float64 `json:"ml_threshold"`
	ResidentialProxyFloor float64 `json:"residential_proxy_floor"`
	ResidentialProxyMaxML float64 `json:"residential_proxy_max_ml"`
	FPGuardCap            float64 `json:"fp_guard_cap"`

	ProxyMinEvents             float64 `json:"proxy_min_events"`
	ProxyMaxCTR                float64 `json:"proxy_max_ctr"`
	ProxyMinUsers              float64 `json:"proxy_min_users"`
	ProxyMinUserClickGap       float64 `json:"proxy_min_user_click_gap"`
	ProxyMinEventsPerUser      float64 `json:"proxy_min_events_per_user"`
	ProxyMinImpressionPressure float64 `json:"proxy_min_impression_pressure"`
	ProxyMinUsersPerUA         float64 `json:"proxy_min_users_per_ua"`
	ProxyMinClicks             float64 `json:"proxy_min_clicks"`

	StructuralHighCTR          float64 `json:"structural_high_ctr"`
	StructuralMaxUsers         float64 `json:"structural_max_users"`
	StructuralMinEvents        float64 `json:"structural_min_events"`
	StructuralMinEventsPerUA     float64 `json:"structural_min_events_per_ua"`
	StructuralMinClicksPerUser float64 `json:"structural_min_clicks_per_user"`
	StructuralSpendRatio       float64 `json:"structural_spend_ratio"`
	StructuralSpendMinCTR      float64 `json:"structural_spend_min_ctr"`
}

func DefaultPolicyConfig() PolicyConfig {
	return PolicyConfig{
		TierPass:    domain.DefaultFraudThresholdPass,
		TierSuspect: domain.DefaultFraudThresholdSuspect,
		TierIVT:     domain.DefaultFraudThresholdIVT,
		TierBlock:   domain.DefaultFraudThresholdBlock,

		MLThreshold:          0.50,
		ResidentialProxyFloor: 0.62,
		ResidentialProxyMaxML: 0.45,
		FPGuardCap:           0.79,

		ProxyMinEvents:            80,
		ProxyMaxCTR:               0.05,
		ProxyMinUsers:             20,
		ProxyMinUserClickGap:      5.0,
		ProxyMinEventsPerUser:     5.0,
		ProxyMinImpressionPressure: 12.0,
		ProxyMinUsersPerUA:        2.5,
		ProxyMinClicks:            2,

		StructuralHighCTR:        0.45,
		StructuralMaxUsers:       5,
		StructuralMinEvents:      50,
		StructuralMinEventsPerUA: 80,
		StructuralMinClicksPerUser: 15,
		StructuralSpendRatio:     0.9,
		StructuralSpendMinCTR:    0.4,
	}
}

var activePolicyConfig = DefaultPolicyConfig()

func SetPolicyConfig(cfg PolicyConfig) {
	activePolicyConfig = cfg
}

func GetPolicyConfig() PolicyConfig {
	return activePolicyConfig
}

func (cfg PolicyConfig) BlockProbability() float64 {
	if cfg.TierBlock > 0 && cfg.TierBlock < 100 {
		return float64(cfg.TierBlock) / 100.0
	}
	if cfg.TierIVT > 0 {
		return float64(cfg.TierIVT) / 100.0
	}
	return 0.80
}

func (cfg PolicyConfig) WithCampaignThresholds(pass, suspect, ivt, block uint8) PolicyConfig {
	out := cfg
	if pass > 0 {
		out.TierPass = pass
	}
	if suspect > 0 {
		out.TierSuspect = suspect
	}
	if ivt > 0 {
		out.TierIVT = ivt
	}
	if block > 0 {
		out.TierBlock = block
	}
	return out
}

type policyMetadata struct {
	Policy PolicyConfig `json:"policy"`
}

func LoadPolicyFromMetadata(path string) (PolicyConfig, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PolicyConfig{}, false
	}
	var meta policyMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return PolicyConfig{}, false
	}
	if meta.Policy == (PolicyConfig{}) {
		return PolicyConfig{}, false
	}
	if meta.Policy.TierPass == 0 && meta.Policy.TierSuspect == 0 {
		return PolicyConfig{}, false
	}
	return meta.Policy, true
}

func ResolvePolicyConfig(envCfg PolicyConfig, metadataPath string, source string) PolicyConfig {
	switch source {
	case "metadata":
		if meta, ok := LoadPolicyFromMetadata(metadataPath); ok {
			return meta
		}
		return envCfg
	case "env":
		return envCfg
	default:
		if meta, ok := LoadPolicyFromMetadata(metadataPath); ok {
			return mergePolicyConfig(meta, envCfg)
		}
		return envCfg
	}
}

func mergePolicyConfig(base, override PolicyConfig) PolicyConfig {
	def := DefaultPolicyConfig()
	out := base
	if override.TierPass != def.TierPass {
		out.TierPass = override.TierPass
	}
	if override.TierSuspect != def.TierSuspect {
		out.TierSuspect = override.TierSuspect
	}
	if override.TierIVT != def.TierIVT {
		out.TierIVT = override.TierIVT
	}
	if override.TierBlock != def.TierBlock {
		out.TierBlock = override.TierBlock
	}
	out = mergePolicyFloats(out, override)
	return out
}

func mergePolicyFloats(out, override PolicyConfig) PolicyConfig {
	def := DefaultPolicyConfig()
	if override.MLThreshold != def.MLThreshold {
		out.MLThreshold = override.MLThreshold
	}
	if override.ResidentialProxyFloor != def.ResidentialProxyFloor {
		out.ResidentialProxyFloor = override.ResidentialProxyFloor
	}
	if override.ResidentialProxyMaxML != def.ResidentialProxyMaxML {
		out.ResidentialProxyMaxML = override.ResidentialProxyMaxML
	}
	if override.FPGuardCap != def.FPGuardCap {
		out.FPGuardCap = override.FPGuardCap
	}
	if override.ProxyMinEvents != def.ProxyMinEvents {
		out.ProxyMinEvents = override.ProxyMinEvents
	}
	if override.ProxyMaxCTR != def.ProxyMaxCTR {
		out.ProxyMaxCTR = override.ProxyMaxCTR
	}
	if override.ProxyMinUsers != def.ProxyMinUsers {
		out.ProxyMinUsers = override.ProxyMinUsers
	}
	if override.ProxyMinUserClickGap != def.ProxyMinUserClickGap {
		out.ProxyMinUserClickGap = override.ProxyMinUserClickGap
	}
	if override.ProxyMinEventsPerUser != def.ProxyMinEventsPerUser {
		out.ProxyMinEventsPerUser = override.ProxyMinEventsPerUser
	}
	if override.ProxyMinImpressionPressure != def.ProxyMinImpressionPressure {
		out.ProxyMinImpressionPressure = override.ProxyMinImpressionPressure
	}
	if override.ProxyMinUsersPerUA != def.ProxyMinUsersPerUA {
		out.ProxyMinUsersPerUA = override.ProxyMinUsersPerUA
	}
	if override.ProxyMinClicks != def.ProxyMinClicks {
		out.ProxyMinClicks = override.ProxyMinClicks
	}
	if override.StructuralHighCTR != def.StructuralHighCTR {
		out.StructuralHighCTR = override.StructuralHighCTR
	}
	if override.StructuralMaxUsers != def.StructuralMaxUsers {
		out.StructuralMaxUsers = override.StructuralMaxUsers
	}
	if override.StructuralMinEvents != def.StructuralMinEvents {
		out.StructuralMinEvents = override.StructuralMinEvents
	}
	if override.StructuralMinEventsPerUA != def.StructuralMinEventsPerUA {
		out.StructuralMinEventsPerUA = override.StructuralMinEventsPerUA
	}
	if override.StructuralMinClicksPerUser != def.StructuralMinClicksPerUser {
		out.StructuralMinClicksPerUser = override.StructuralMinClicksPerUser
	}
	if override.StructuralSpendRatio != def.StructuralSpendRatio {
		out.StructuralSpendRatio = override.StructuralSpendRatio
	}
	if override.StructuralSpendMinCTR != def.StructuralSpendMinCTR {
		out.StructuralSpendMinCTR = override.StructuralSpendMinCTR
	}
	return out
}

func PolicyConfigFromEnv() PolicyConfig {
	def := DefaultPolicyConfig()
	cfg := def
	cfg.TierPass = uint8(envInt("FRAUD_POLICY_TIER_PASS", int(def.TierPass)))
	cfg.TierSuspect = uint8(envInt("FRAUD_POLICY_TIER_SUSPECT", int(def.TierSuspect)))
	cfg.TierIVT = uint8(envInt("FRAUD_POLICY_TIER_IVT", int(def.TierIVT)))
	cfg.TierBlock = uint8(envInt("FRAUD_POLICY_TIER_BLOCK", int(def.TierBlock)))

	cfg.MLThreshold = envFloat("FRAUD_POLICY_ML_THRESHOLD", def.MLThreshold)
	cfg.ResidentialProxyFloor = envFloat("FRAUD_POLICY_PROXY_FLOOR", def.ResidentialProxyFloor)
	cfg.ResidentialProxyMaxML = envFloat("FRAUD_POLICY_PROXY_MAX_ML", def.ResidentialProxyMaxML)
	cfg.FPGuardCap = envFloat("FRAUD_POLICY_FP_GUARD_CAP", def.FPGuardCap)

	cfg.ProxyMinEvents = envFloat("FRAUD_POLICY_PROXY_MIN_EVENTS", def.ProxyMinEvents)
	cfg.ProxyMaxCTR = envFloat("FRAUD_POLICY_PROXY_MAX_CTR", def.ProxyMaxCTR)
	cfg.ProxyMinUsers = envFloat("FRAUD_POLICY_PROXY_MIN_USERS", def.ProxyMinUsers)
	cfg.ProxyMinUserClickGap = envFloat("FRAUD_POLICY_PROXY_MIN_USER_CLICK_GAP", def.ProxyMinUserClickGap)
	cfg.ProxyMinEventsPerUser = envFloat("FRAUD_POLICY_PROXY_MIN_EVENTS_PER_USER", def.ProxyMinEventsPerUser)
	cfg.ProxyMinImpressionPressure = envFloat("FRAUD_POLICY_PROXY_MIN_IMPRESSION_PRESSURE", def.ProxyMinImpressionPressure)
	cfg.ProxyMinUsersPerUA = envFloat("FRAUD_POLICY_PROXY_MIN_USERS_PER_UA", def.ProxyMinUsersPerUA)
	cfg.ProxyMinClicks = envFloat("FRAUD_POLICY_PROXY_MIN_CLICKS", def.ProxyMinClicks)

	cfg.StructuralHighCTR = envFloat("FRAUD_POLICY_STRUCTURAL_HIGH_CTR", def.StructuralHighCTR)
	cfg.StructuralMaxUsers = envFloat("FRAUD_POLICY_STRUCTURAL_MAX_USERS", def.StructuralMaxUsers)
	cfg.StructuralMinEvents = envFloat("FRAUD_POLICY_STRUCTURAL_MIN_EVENTS", def.StructuralMinEvents)
	cfg.StructuralMinEventsPerUA = envFloat("FRAUD_POLICY_STRUCTURAL_MIN_EVENTS_PER_UA", def.StructuralMinEventsPerUA)
	cfg.StructuralMinClicksPerUser = envFloat("FRAUD_POLICY_STRUCTURAL_MIN_CLICKS_PER_USER", def.StructuralMinClicksPerUser)
	cfg.StructuralSpendRatio = envFloat("FRAUD_POLICY_STRUCTURAL_SPEND_RATIO", def.StructuralSpendRatio)
	cfg.StructuralSpendMinCTR = envFloat("FRAUD_POLICY_STRUCTURAL_SPEND_MIN_CTR", def.StructuralSpendMinCTR)
	return cfg
}

func envFloat(key string, fallback float64) float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return v
}

func envInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}
