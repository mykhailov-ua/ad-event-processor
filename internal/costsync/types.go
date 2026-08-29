package costsync

import "ad-event-processor/internal/costsync/provider"

type LineType = provider.LineType

const (
	LineTypeSpend   = provider.LineTypeSpend
	LineTypeRevenue = provider.LineTypeRevenue
)

const (
	AttributionModeToken  = provider.AttributionModeToken
	AttributionModeSpread = provider.AttributionModeSpread
)

type (
	CostLine     = provider.CostLine
	Credential   = provider.Credential
	TokenMapping = provider.TokenMapping
)

var (
	ParseTokenMapping        = provider.ParseTokenMapping
	ValidSyncIntervalMinutes = provider.ValidSyncIntervalMinutes
)
