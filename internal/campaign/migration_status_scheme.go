package campaign

import "ad-event-processor/internal/migrationsource"

func statusSchemeRulesFromMigration(rules []migrationsource.MappedStatusSchemeRule) []StatusSchemeRuleDTO {
	out := make([]StatusSchemeRuleDTO, 0, len(rules))
	for _, row := range rules {
		out = append(out, StatusSchemeRuleDTO{
			SortOrder:         row.SortOrder,
			WhenStatus:        row.WhenStatus,
			WhenGoal:          row.WhenGoal,
			SetInternalStatus: row.SetInternalStatus,
			SetGoalName:       row.SetGoalName,
			PayoutMode:        row.PayoutMode,
			PayoutMicro:       row.PayoutMicro,
			FireOutbound:      row.FireOutbound,
			Enabled:           row.Enabled,
		})
	}
	return out
}
