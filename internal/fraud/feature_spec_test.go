package fraud

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type featureFixture struct {
	ID           string     `json:"id"`
	FeatureNames []string   `json:"feature_names"`
	Row          fixtureRow `json:"row"`
	Vector       []float64  `json:"vector"`
}

type fixtureRow struct {
	Events           uint64 `json:"events"`
	Clicks           uint64 `json:"clicks"`
	SpendMicro       int64  `json:"spend_micro"`
	BudgetLimitMicro int64  `json:"budget_limit_micro"`
	UniqueUsers      uint64 `json:"unique_users"`
	UniqueUAs        uint64 `json:"unique_uas"`
}

func TestFeatureSpecDims(t *testing.T) {
	assert.Equal(t, 16, Dims())
	assert.Len(t, FeatureNames, Dims())
}

func TestFeatureSpecGoldenFixtures(t *testing.T) {
	root := repoRoot(t)
	fixturesDir := filepath.Join(root, "testdata", "ml")
	if _, err := os.Stat(fixturesDir); os.IsNotExist(err) {
		t.Skip("testdata/ml fixtures not present; run make fraudtrain-check locally")
	}
	entries, err := os.ReadDir(fixturesDir)
	require.NoError(t, err)

	var loaded int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "features_") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(fixturesDir, entry.Name())
		data, err := os.ReadFile(path)
		require.NoError(t, err, entry.Name())

		var fixture featureFixture
		require.NoError(t, json.Unmarshal(data, &fixture), entry.Name())
		require.Equal(t, FeatureNames, fixture.FeatureNames, entry.Name())

		row := FeatureRow{
			Events:           fixture.Row.Events,
			Clicks:           fixture.Row.Clicks,
			SpendMicro:       fixture.Row.SpendMicro,
			BudgetLimitMicro: fixture.Row.BudgetLimitMicro,
			UniqueUsers:      fixture.Row.UniqueUsers,
			UniqueUAs:        fixture.Row.UniqueUAs,
		}
		got := row.ToVector()
		require.Len(t, got, Dims(), entry.Name())
		for i := range got {
			assert.InDelta(t, fixture.Vector[i], got[i], 1e-9, "%s[%d]", entry.Name(), i)
		}
		loaded++
	}
	if loaded == 0 {
		t.Skip("no features_*.json fixtures under var/fraudscore/fixtures; run make fraudtrain-check locally")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
