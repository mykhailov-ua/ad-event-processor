package fraudscoring

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLGBMScorer(t *testing.T) {
	scorer, err := NewLGBMScorer(testModelPath(t))
	if err != nil {
		t.Fatalf("failed to create LGBMScorer: %v", err)
	}

	assert.Equal(t, "lightgbm", scorer.Name())
	assert.Equal(t, 16, scorer.Dims())

	rows := []FeatureRow{
		{
			WindowStart:      time.Now(),
			IPAddress:        "1.2.3.4",
			CampaignID:       "00000000-0000-0000-0000-000000000001",
			Events:           10,
			Clicks:           2,
			SpendMicro:       1000000,
			BudgetLimitMicro: 5000000,
			UniqueUsers:      1,
			UniqueUAs:        1,
		},
		{
			WindowStart:      time.Now(),
			IPAddress:        "5.6.7.8",
			CampaignID:       "00000000-0000-0000-0000-000000000002",
			Events:           100,
			Clicks:           10,
			SpendMicro:       10000000,
			BudgetLimitMicro: 50000000,
			UniqueUsers:      5,
			UniqueUAs:        2,
		},
	}

	scores, err := scorer.ScoreBatch(context.Background(), rows)
	if err != nil {
		t.Fatalf("ScoreBatch failed: %v", err)
	}

	assert.Len(t, scores, 2)
	for _, score := range scores {
		assert.GreaterOrEqual(t, score, 0.0)
		assert.LessOrEqual(t, score, 1.0)
	}
	assert.Greater(t, scores[1], scores[0])
}

func TestEnsembleScorer(t *testing.T) {
	modelPath := testModelPath(t)
	scorer1, err := NewLGBMScorer(modelPath)
	if err != nil {
		t.Fatalf("failed to create scorer1: %v", err)
	}

	scorer2, err := NewLGBMScorer(modelPath)
	if err != nil {
		t.Fatalf("failed to create scorer2: %v", err)
	}

	ensemble := NewEnsemble(scorer1, scorer2)
	assert.Equal(t, "ensemble", ensemble.Name())
	assert.Equal(t, 16, ensemble.Dims())

	rows := []FeatureRow{
		{
			WindowStart:      time.Now(),
			IPAddress:        "1.2.3.4",
			CampaignID:       "00000000-0000-0000-0000-000000000001",
			Events:           10,
			Clicks:           2,
			SpendMicro:       1000000,
			BudgetLimitMicro: 5000000,
			UniqueUsers:      1,
			UniqueUAs:        1,
		},
	}

	scores, err := ensemble.ScoreBatch(context.Background(), rows)
	if err != nil {
		t.Fatalf("ScoreBatch failed: %v", err)
	}

	assert.Len(t, scores, 1)
	assert.GreaterOrEqual(t, scores[0], 0.0)
	assert.LessOrEqual(t, scores[0], 1.0)
}
