package ingestion

import _ "embed"

//go:embed decoy_unified_filter.lua
var decoyUnifiedFilterLua string

func decoyUnifiedFilterEmbed() string {
	return decoyUnifiedFilterLua
}
