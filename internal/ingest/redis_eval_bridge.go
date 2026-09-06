package ingest

import (
	"ad-event-processor/internal/filter"
	filterunified "ad-event-processor/internal/filter/unified"

	"github.com/redis/go-redis/v9"
)

type redisCmdHead = filterunified.RedisCmdHead

const unifiedFilterKeyCount = filterunified.UnifiedFilterKeyCount

var (
	numKeys15Any        = filterunified.NumKeys15Any
	resetPooledRedisCmd = filterunified.ResetPooledRedisCmd
	isNoScriptErr       = filterunified.IsNoScriptErr
)

func evalCmdPoolGet() *redis.Cmd {
	return filterunified.EvalCmdPoolGet()
}

func evalCmdPoolPut(cmd *redis.Cmd) {
	filterunified.EvalCmdPoolPut(cmd)
}

var (
	pickGlobalReadShard            = filter.PickGlobalReadShard
	pickLocalGlobalShard           = filter.PickLocalGlobalShard
	pickGlobalReadShardForCampaign = filter.PickGlobalReadShardForCampaign
	pickGlobalReadShardForIP       = filter.PickGlobalReadShardForIP
)
