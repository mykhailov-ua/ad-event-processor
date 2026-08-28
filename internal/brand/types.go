package brand

import (
	"ad-event-processor/internal/controlplane/authz"
)

type DTO struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	FreqLimit  int32  `json:"freq_limit"`
	FreqWindow int32  `json:"freq_window"`
}

type CreativeDTO struct {
	ID         string `json:"id"`
	BrandID    string `json:"brand_id"`
	Name       string `json:"name"`
	LandingURL string `json:"landing_url"`
	Weight     int32  `json:"weight"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

func (c CreativeDTO) Scrub(level authz.MaskLevel) CreativeDTO {
	if level == authz.MaskFull {
		return c
	}
	out := c
	out.LandingURL = ""
	return out
}

type CreateRequest struct {
	CustomerID string `json:"customer_id"`
	Name       string `json:"name"`
}

type UpsertCreativeRequest struct {
	Name       string `json:"name"`
	LandingURL string `json:"landing_url"`
	Weight     int32  `json:"weight"`
	Status     string `json:"status"`
}

type UpdateCreativeRequest struct {
	Name       string `json:"name"`
	LandingURL string `json:"landing_url"`
	Weight     int32  `json:"weight"`
	Status     string `json:"status"`
}

type createdIDResponse struct {
	ID string `json:"id"`
}
