package management

import (
	"fmt"
	"time"

	db "espx/internal/ingestion/sqlc"
)

// validateDaypartHours rejects hour values outside the 0-23 delivery window range.
func validateDaypartHours(hours []int16) error {
	for _, h := range hours {
		if h < 0 || h > 23 {
			return fmt.Errorf("daypart hour must be 0-23, got %d", h)
		}
	}
	return nil
}

// validateSchedule ensures scheduled campaigns have a coherent start and end interval.
func validateSchedule(startAt, endAt *time.Time) error {
	if startAt != nil && endAt != nil && !endAt.After(*startAt) {
		return fmt.Errorf("end_at must be after start_at")
	}
	return nil
}

// countriesOrEmpty normalizes nil country slices to empty JSON arrays in API responses.
func countriesOrEmpty(c []string) []string {
	if c == nil {
		return []string{}
	}
	return c
}

// resolveScheduleStatus derives ACTIVE or PAUSED from whether now falls inside the campaign window.
func resolveScheduleStatus(now time.Time, startAt, endAt *time.Time) db.CampaignStatusType {
	if startAt != nil && now.Before(*startAt) {
		return db.CampaignStatusTypePAUSED
	}
	if endAt != nil && !now.Before(*endAt) {
		return db.CampaignStatusTypePAUSED
	}
	return db.CampaignStatusTypeACTIVE
}
