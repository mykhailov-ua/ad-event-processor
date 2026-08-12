package controlplane

import (
	"time"

	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/internal/controlplane/authz"

	"github.com/jackc/pgx/v5/pgtype"
)

type CampaignCreateSpec = adminapi.CreateCampaignInput

type CampaignTemplateDTO struct {
	ID              string   `json:"id"`
	CustomerID      string   `json:"customer_id"`
	Name            string   `json:"name"`
	BudgetLimit     string   `json:"budget_limit"`
	PacingMode      string   `json:"pacing_mode"`
	DailyBudget     string   `json:"daily_budget"`
	Timezone        string   `json:"timezone"`
	FreqLimit       int32    `json:"freq_limit"`
	FreqWindow      int32    `json:"freq_window"`
	TargetCountries []string `json:"target_countries"`
	BrandID         string   `json:"brand_id,omitempty"`
	DaypartHours    []int16  `json:"daypart_hours"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type BrandCreativeDTO struct {
	ID         string `json:"id"`
	BrandID    string `json:"brand_id"`
	Name       string `json:"name"`
	LandingURL string `json:"landing_url"`
	Weight     int32  `json:"weight"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

func (c BrandCreativeDTO) Scrub(level authz.MaskLevel) BrandCreativeDTO {
	if level == authz.MaskFull {
		return c
	}
	out := c
	out.LandingURL = ""
	return out
}

func toTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
