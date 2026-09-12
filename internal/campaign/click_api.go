package campaign

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

const programmaticClickIDMaxLen = 128

type ProgrammaticClickMintRequest struct {
	CampaignID string            `json:"campaign_id"`
	ClickID    string            `json:"click_id,omitempty"`
	Params     map[string]string `json:"params,omitempty"`
}

type ProgrammaticClickMintResponse struct {
	ClickID  string `json:"click_id"`
	ClickURL string `json:"click_url"`
}

func ValidateProgrammaticClickMintRequest(req ProgrammaticClickMintRequest) (uuid.UUID, error) {
	rawCamp := strings.TrimSpace(req.CampaignID)
	if rawCamp == "" {
		return uuid.Nil, ErrValidationf("campaign_id is required")
	}
	campaignID, err := uuid.Parse(rawCamp)
	if err != nil {
		return uuid.Nil, ErrValidationf("invalid campaign_id")
	}
	clickID := strings.TrimSpace(req.ClickID)
	if clickID != "" && len(clickID) > programmaticClickIDMaxLen {
		return uuid.Nil, ErrValidationf("click_id too long")
	}
	return campaignID, nil
}

func ResolveProgrammaticClickID(provided string) (string, error) {
	clickID := strings.TrimSpace(provided)
	if clickID != "" {
		return clickID, nil
	}
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate click_id: %w", err)
	}
	return id.String(), nil
}

func BuildCampaignClickURL(base string, campaignID uuid.UUID, clickID string, extra map[string]string) (string, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return "", errValidation("tracker public base URL is not configured")
	}
	clickID = strings.TrimSpace(clickID)
	if clickID == "" {
		return "", errValidation("click_id is required")
	}
	u, err := url.Parse(base + "/click")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("campaign_id", campaignID.String())
	q.Set("click_id", clickID)
	for key, value := range extra {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func MintProgrammaticClick(
	base string,
	campaignID uuid.UUID,
	req ProgrammaticClickMintRequest,
	signing ProgrammaticClickLinkSigning,
) (ProgrammaticClickMintResponse, error) {
	clickID, err := ResolveProgrammaticClickID(req.ClickID)
	if err != nil {
		return ProgrammaticClickMintResponse{}, err
	}
	clickURL, err := BuildCampaignClickURL(base, campaignID, clickID, req.Params)
	if err != nil {
		return ProgrammaticClickMintResponse{}, err
	}
	clickURL = MaybeSignProgrammaticClickURL(clickURL, clickID, signing)
	return ProgrammaticClickMintResponse{
		ClickID:  clickID,
		ClickURL: clickURL,
	}, nil
}
