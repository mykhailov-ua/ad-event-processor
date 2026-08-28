package campaign

import (
	"time"

	"github.com/google/uuid"
)

type CreateCampaignSpec struct {
	CustomerID       uuid.UUID
	BrandID          *uuid.UUID
	Name             string
	BudgetLimitMicro int64
	PacingMode       string
	DailyBudgetMicro int64
	Timezone         string
	FreqLimit        int32
	FreqWindow       int32
	TargetCountries  []string
	StartAt          *time.Time
	EndAt            *time.Time
	DaypartHours     []int16
	TemplateID       *uuid.UUID
	IdempotencyKey   string
}
