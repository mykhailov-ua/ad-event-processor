package controlplane

import (
	"testing"

	"ad-event-processor/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestCalculateOverdraft_reconLagPenalty(t *testing.T) {
	t.Parallel()
	cfg := configWithReconPenalty()
	worker := &CreditScoringWorker{svc: &Service{cfg: cfg}}
	base := worker.calculateOverdraft(40, 1_000_000_000, 0)
	penalized := worker.calculateOverdraft(40, 1_000_000_000, 500_000_000)
	assert.Equal(t, base/2, penalized)
}

func configWithReconPenalty() *config.Config {
	cfg := &config.Config{
		CreditScoringMinAgeDays:         7,
		CreditScoringMatureAgeDays:      30,
		CreditScoringMaturePercent:      30,
		CreditScoringMaxCap:             10_000_000_000,
		CreditScoringReconLagThreshold:  100_000_000,
		CreditScoringReconLagPenaltyPct: 50,
	}
	return cfg
}
