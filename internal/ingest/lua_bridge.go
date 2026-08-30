package ingest

import (
	"ad-event-processor/internal/domain"
	filterunified "ad-event-processor/internal/filter/unified"
	"ad-event-processor/internal/track"
)

var (
	unifiedFilterLua              = filterunified.EmbeddedUnifiedFilterLua()
	budgetFastLua                 = filterunified.BudgetFastLua()
	resolveUnifiedFilterLuaSource = filterunified.ResolveUnifiedFilterLuaSource
)

func fcapKeyPrefixForDebit(camp *domain.Campaign, userID, clickID string) string {
	return filterunified.FcapKeyPrefixForDebit(camp, userID, clickID)
}

var safePageHydratorJS = track.SafePageHydratorJS()

const (
	luaReturnTierDegraded = filterunified.LuaReturnTierDegraded
	luaReturnFraudSignal  = filterunified.LuaReturnFraudSignal
)

func luaBranchLabel(res int64) string {
	return filterunified.LuaBranchLabel(res)
}

const (
	localTTCLow                   = filterunified.LocalTTCLow
	localTTCBypass                = filterunified.LocalTTCBypass
	localTTCMissingClosed         = filterunified.LocalTTCMissingClosed
	sealedUnifiedFilterAssetLabel = filterunified.SealedUnifiedFilterAssetLabel
)
