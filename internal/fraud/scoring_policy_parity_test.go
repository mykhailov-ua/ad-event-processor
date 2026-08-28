package fraud

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type policyParityFile struct {
	Cases []policyParityCase `json:"cases"`
}

type policyParityCase struct {
	ID            string          `json:"id"`
	Op            string          `json:"op"`
	Row           fixtureRow      `json:"row"`
	MLProbability float64         `json:"ml_probability"`
	Want          json.RawMessage `json:"want"`
}

type policyAdjustWant struct {
	AdjustedProbability float64 `json:"adjusted_probability"`
	ResidentialProxy    bool    `json:"residential_proxy"`
	StructuralFraud     bool    `json:"structural_fraud"`
	FPGuardApplied      bool    `json:"fp_guard_applied"`
}

type policyDecideWant struct {
	Tier                string  `json:"tier"`
	Score               int     `json:"score"`
	AdjustedProbability float64 `json:"adjusted_probability"`
	ResidentialProxy    bool    `json:"residential_proxy"`
	StructuralFraud     bool    `json:"structural_fraud"`
	FPGuardApplied      bool    `json:"fp_guard_applied"`
}

func TestScoringPolicyParityFixtures(t *testing.T) {
	fixturesDir := fraudMLFixtureDir(t)
	path := filepath.Join(fixturesDir, "policy_parity.json")
	require.FileExists(t, path, "run python3 -m train.fixture_generator if missing")

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var file policyParityFile
	require.NoError(t, json.Unmarshal(data, &file))
	require.NotEmpty(t, file.Cases)

	for _, tc := range file.Cases {
		t.Run(tc.ID, func(t *testing.T) {
			row := featureRowFromFixture(tc.Row)
			switch tc.Op {
			case "residential_proxy_signal":
				var want bool
				require.NoError(t, json.Unmarshal(tc.Want, &want))
				assert.Equal(t, want, ResidentialProxySignal(row))
			case "adjust_probability":
				var want policyAdjustWant
				require.NoError(t, json.Unmarshal(tc.Want, &want))
				adjusted, proxy, structural, fpGuard := AdjustProbability(row, tc.MLProbability)
				assert.InDelta(t, want.AdjustedProbability, adjusted, 1e-9)
				assert.Equal(t, want.ResidentialProxy, proxy)
				assert.Equal(t, want.StructuralFraud, structural)
				assert.Equal(t, want.FPGuardApplied, fpGuard)
			case "decide":
				var want policyDecideWant
				require.NoError(t, json.Unmarshal(tc.Want, &want))
				decision := Decide(row, tc.MLProbability)
				assert.Equal(t, FraudTier(want.Tier), decision.Tier)
				assert.Equal(t, want.Score, decision.Score)
				assert.InDelta(t, want.AdjustedProbability, decision.AdjustedProbability, 1e-9)
				assert.Equal(t, want.ResidentialProxy, decision.ResidentialProxy)
				assert.Equal(t, want.StructuralFraud, decision.StructuralFraud)
				assert.Equal(t, want.FPGuardApplied, decision.FPGuardApplied)
			default:
				t.Fatalf("unknown op %q", tc.Op)
			}
		})
	}
}

func featureRowFromFixture(row fixtureRow) FeatureRow {
	return FeatureRow{
		Events:           row.Events,
		Clicks:           row.Clicks,
		SpendMicro:       row.SpendMicro,
		BudgetLimitMicro: row.BudgetLimitMicro,
		UniqueUsers:      row.UniqueUsers,
		UniqueUAs:        row.UniqueUAs,
	}
}
