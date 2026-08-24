package fraud

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMicrobatchBoostScore_suspectTierOnly(t *testing.T) {
	row := FeatureRow{
		WindowStart: time.Now(),
		IPAddress:   "1.2.3.4",
		CampaignID:  "00000000-0000-0000-0000-000000000001",
		Events:      10,
		Clicks:      2,
		UniqueUsers: 1,
		UniqueUAs:   1,
	}

	score, ok := microbatchBoostScore(row, 0.10)
	assert.False(t, ok)
	assert.Equal(t, 0, score)

	score, ok = microbatchBoostScore(row, 0.55)
	assert.True(t, ok)
	assert.Greater(t, score, 0)
}

func TestScoreBoostTTL_matchesOutbox(t *testing.T) {
	assert.Equal(t, 900*time.Second, ScoreBoostTTL)
}
