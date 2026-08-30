package trafficoptimizer

import (
	"encoding/json"
	"time"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

type Rule struct {
	ID                  uuid.UUID
	CustomerID          uuid.UUID
	CampaignID          uuid.UUID
	HasCampaign         bool
	FlowID              uuid.UUID
	HasFlow             bool
	BrandID             uuid.UUID
	HasBrand            bool
	Name                string
	Scope               string
	Objective           string
	Algorithm           string
	LookbackMinutes     int
	MinClicks           int
	MinConversions      int
	MinSpendMicro       int64
	EvalIntervalMinutes int
	CooldownMinutes     int
	MaxWeightDeltaPct   int
	Enabled             bool
	LastEvaluatedAt     time.Time
	HasLastEvaluated    bool
}

func RuleFromRow(row db.TrafficOptimizerRule) (Rule, error) {
	rule := Rule{
		ID:                  uuid.UUID(row.ID.Bytes),
		CustomerID:          uuid.UUID(row.CustomerID.Bytes),
		Name:                row.Name,
		Scope:               row.Scope,
		Objective:           row.Objective,
		Algorithm:           row.Algorithm,
		LookbackMinutes:     int(row.LookbackMinutes),
		MinClicks:           int(row.MinClicks),
		MinConversions:      int(row.MinConversions),
		MinSpendMicro:       row.MinSpendMicro,
		EvalIntervalMinutes: int(row.EvalIntervalMinutes),
		CooldownMinutes:     int(row.CooldownMinutes),
		MaxWeightDeltaPct:   int(row.MaxWeightDeltaPct),
		Enabled:             row.Enabled,
	}
	if rule.EvalIntervalMinutes <= 0 {
		rule.EvalIntervalMinutes = 15
	}
	if row.CampaignID.Valid {
		rule.CampaignID = uuid.UUID(row.CampaignID.Bytes)
		rule.HasCampaign = true
	}
	if row.FlowID.Valid {
		rule.FlowID = uuid.UUID(row.FlowID.Bytes)
		rule.HasFlow = true
	}
	if row.BrandID.Valid {
		rule.BrandID = uuid.UUID(row.BrandID.Bytes)
		rule.HasBrand = true
	}
	if row.LastEvaluatedAt.Valid {
		rule.LastEvaluatedAt = row.LastEvaluatedAt.Time
		rule.HasLastEvaluated = true
	}
	return rule, nil
}

func encodePresetParameters(params map[string]float64) ([]byte, error) {
	if len(params) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(params)
}
