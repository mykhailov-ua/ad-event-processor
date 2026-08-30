package ingest

import (
	"ad-event-processor/internal/filter"
	filterunified "ad-event-processor/internal/filter/unified"
)

type redisCmdHead = filterunified.RedisCmdHead

const unifiedFilterKeyCount = filterunified.UnifiedFilterKeyCount

var (
	numKeys15Any        = filterunified.NumKeys15Any
	evalCmdPool         = filterunified.EvalCmdPool
	resetPooledRedisCmd = filterunified.ResetPooledRedisCmd
	isNoScriptErr       = filterunified.IsNoScriptErr
)

var (
	pickGlobalReadShard            = filter.PickGlobalReadShard
	pickLocalGlobalShard           = filter.PickLocalGlobalShard
	pickGlobalReadShardForCampaign = filter.PickGlobalReadShardForCampaign
	pickGlobalReadShardForIP       = filter.PickGlobalReadShardForIP
)
