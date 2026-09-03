package dashboardadmin

import (
	"fmt"
	"strings"
)

type CampaignReportDimension string

const (
	CampaignReportDimPaths      CampaignReportDimension = "paths"
	CampaignReportDimOffers     CampaignReportDimension = "offers"
	CampaignReportDimLanders    CampaignReportDimension = "landers"
	CampaignReportDimRules      CampaignReportDimension = "rules"
	CampaignReportDimTokens     CampaignReportDimension = "tokens"
	CampaignReportDimConnection CampaignReportDimension = "connection"
	CampaignReportDimDevice     CampaignReportDimension = "device"
	CampaignReportDimCountry    CampaignReportDimension = "country"
	CampaignReportDimDefault    CampaignReportDimension = "default"
)

var allowedCampaignReportDimensions = map[CampaignReportDimension]struct{}{
	CampaignReportDimPaths:      {},
	CampaignReportDimOffers:     {},
	CampaignReportDimLanders:    {},
	CampaignReportDimRules:      {},
	CampaignReportDimTokens:     {},
	CampaignReportDimConnection: {},
	CampaignReportDimDevice:     {},
	CampaignReportDimCountry:    {},
	CampaignReportDimDefault:    {},
}

func ParseCampaignReportDimension(raw string) (CampaignReportDimension, error) {
	dimension := CampaignReportDimension(strings.TrimSpace(strings.ToLower(raw)))
	if dimension == "" {
		return CampaignReportDimCountry, nil
	}
	if _, ok := allowedCampaignReportDimensions[dimension]; !ok {
		return "", fmt.Errorf("invalid dimension")
	}
	return dimension, nil
}
