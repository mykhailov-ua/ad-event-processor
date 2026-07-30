package management

import (
	"fmt"
	"time"

	db "espx/internal/ingestion/sqlc"
)

func validateDaypartHours(hours []int16) error {
	for _, h := range hours {
		if h < 0 || h > 23 {
			return fmt.Errorf("daypart hour must be 0-23, got %d", h)
		}
	}
	return nil
}

func validateSchedule(startAt, endAt *time.Time) error {
	if startAt != nil && endAt != nil && !endAt.After(*startAt) {
		return fmt.Errorf("end_at must be after start_at")
	}
	return nil
}

func countriesOrEmpty(c []string) []string {
	if c == nil {
		return []string{}
	}
	return c
}

func resolveScheduleStatus(now time.Time, startAt, endAt *time.Time) db.CampaignStatusType {
	if startAt != nil && now.Before(*startAt) {
		return db.CampaignStatusTypePAUSED
	}
	if endAt != nil && !now.Before(*endAt) {
		return db.CampaignStatusTypePAUSED
	}
	return db.CampaignStatusTypeACTIVE
}
