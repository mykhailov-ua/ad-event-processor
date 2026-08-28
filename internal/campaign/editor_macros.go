package campaign

import (
	"strings"
)

type MacroPreviewRequestDTO struct {
	Sub1    string `json:"sub1,omitempty"`
	Country string `json:"country,omitempty"`
	ClickID string `json:"click_id,omitempty"`
	UserID  string `json:"user_id,omitempty"`
	FBCLID  string `json:"fbclid,omitempty"`
	GCLID   string `json:"gclid,omitempty"`
	TTCLID  string `json:"ttclid,omitempty"`
}

type MacroPreviewResponseDTO struct {
	ResolvedClickURL    string   `json:"resolved_click_url,omitempty"`
	ResolvedPostbackURL string   `json:"resolved_postback_url,omitempty"`
	UnresolvedMacros    []string `json:"unresolved_macros,omitempty"`
	Warnings            []string `json:"warnings,omitempty"`
}

func previewCampaignMacros(campaign CampaignDTO, req MacroPreviewRequestDTO, masked bool) (MacroPreviewResponseDTO, error) {
	baseURL := campaign.TargetURL
	if masked {
		baseURL = "[redacted-offer-url]"
	}
	if strings.TrimSpace(baseURL) == "" {
		return MacroPreviewResponseDTO{}, errValidation("target_url is required for macro preview")
	}
	macroCtx := PreviewContext(campaign.ID, PreviewRequest{
		Sub1:    req.Sub1,
		Country: req.Country,
		ClickID: req.ClickID,
		UserID:  req.UserID,
		FBCLID:  req.FBCLID,
		GCLID:   req.GCLID,
		TTCLID:  req.TTCLID,
	})
	resolved, unresolved := Expand(baseURL, macroCtx)
	if params := campaign.ClickQueryParams; len(params) > 0 {
		var parts []string
		for k, v := range params {
			expanded, paramUnresolved := Expand(v, macroCtx)
			unresolved = append(unresolved, paramUnresolved...)
			parts = append(parts, k+"="+expanded)
		}
		if len(parts) > 0 && !strings.Contains(resolved, "?") {
			resolved += "?" + strings.Join(parts, "&")
		}
	}
	var warnings []string
	if strings.HasPrefix(strings.ToLower(resolved), "http://") {
		warnings = append(warnings, "click url uses http")
	}
	return MacroPreviewResponseDTO{
		ResolvedClickURL: resolved,
		UnresolvedMacros: unresolved,
		Warnings:         warnings,
	}, nil
}

func CampaignRevision(updatedAt string) string {
	return strings.TrimSpace(updatedAt)
}

func campaignRevision(updatedAt string) string {
	return CampaignRevision(updatedAt)
}

type CampaignConflictResponseDTO struct {
	Error          string      `json:"error"`
	ServerRevision string      `json:"server_revision"`
	ConflictFields []string    `json:"conflict_fields,omitempty"`
	MergeHintLabel string      `json:"merge_hint_label,omitempty"`
	Current        CampaignDTO `json:"current"`
}
