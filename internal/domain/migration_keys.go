package domain

import "github.com/google/uuid"

const (
	MigrationFenceKeyPrefix = "budget:migration_fence:"
	BudgetFrozenKeyPrefix   = "budget:frozen:"
)

func MigrationFenceRedisKey(campaignID uuid.UUID) string {
	return MigrationFenceKeyPrefix + campaignID.String()
}

func BudgetFrozenRedisKey(campaignID uuid.UUID) string {
	return BudgetFrozenKeyPrefix + campaignID.String()
}
