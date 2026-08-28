package campaign

import (
	"fmt"
	"time"

	db "ad-event-processor/internal/domain/db"
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

func ResolveScheduleStatus(now time.Time, startAt, endAt *time.Time) db.CampaignStatusType {
	if startAt != nil && now.Before(*startAt) {
		return db.CampaignStatusTypePAUSED
	}
	if endAt != nil && !now.Before(*endAt) {
		return db.CampaignStatusTypePAUSED
	}
	return db.CampaignStatusTypeACTIVE
}

func resolveScheduleStatus(now time.Time, startAt, endAt *time.Time) db.CampaignStatusType {
	return ResolveScheduleStatus(now, startAt, endAt)
}

func ValidateDaypartHours(hours []int16) error {
	return validateDaypartHours(hours)
}

func ValidateSchedule(startAt, endAt *time.Time) error {
	return validateSchedule(startAt, endAt)
}

func CountriesOrEmpty(c []string) []string {
	return countriesOrEmpty(c)
}

func DaypartOrEmpty(hours []int16) []int16 {
	if hours == nil {
		return []int16{}
	}
	return hours
}
