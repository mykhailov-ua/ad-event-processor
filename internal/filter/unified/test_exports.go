package unified

import (
	"context"

	"ad-event-processor/internal/domain"

	"github.com/redis/go-redis/v9"
)

type RedisCmdHead = redisCmdHead

const UnifiedFilterKeyCount = unifiedFilterKeyCount

var (
	NumKeys15Any = numKeys15Any
	EvalCmdPool  = evalCmdPool
)

func ResetPooledRedisCmd(cmd *redis.Cmd, ctx context.Context, args []any, firstKeyPos int8) {
	resetPooledRedisCmd(cmd, ctx, args, firstKeyPos)
}

type FilterEvalPinSlot = filterEvalPinSlot

func (s *FilterEvalPinSlot) Conn() *redis.Conn {
	if s == nil {
		return nil
	}
	return s.conn
}

func (f *UnifiedFilter) EvalPinConn(evt *domain.Event, shard int) *redis.Conn {
	return f.evalPinConn(evt, shard)
}

func (f *UnifiedFilter) EvalPinSlot(worker, shard int) *FilterEvalPinSlot {
	if f == nil || f.evalPins == nil {
		return nil
	}
	return f.evalPins.slot(worker, shard)
}

func (f *UnifiedFilter) EvalShaPooled(
	ctx context.Context,
	c redis.UniversalClient,
	shard int,
	evt *domain.Event,
	sha1 any,
	keyArgs [UnifiedFilterKeyCount]any,
	scriptArgs []any,
) (int64, error) {
	return f.evalShaPooled(ctx, c, shard, evt, sha1, keyArgs, scriptArgs)
}

func IsNoScriptErr(err error) bool {
	return isNoScriptErr(err)
}

func (f *UnifiedFilter) PreloadScriptsShard(ctx context.Context, shard int, redisClient redis.UniversalClient) error {
	return f.preloadScriptsShard(ctx, shard, redisClient)
}

func ParseRedisUsedMemory(info string) int64 {
	return parseRedisUsedMemory(info)
}

var (
	OneAny  = oneAny
	ZeroAny = zeroAny
)

func FirstConnectedRedisShard(redisShards []redis.UniversalClient) redis.UniversalClient {
	return firstConnectedRedisShard(redisShards)
}

func TCPMSSWireValue(mss uint16) uint16 {
	return tcpMSSWireValue(mss)
}

const SealedUnifiedFilterAssetLabel = sealedUnifiedFilterAssetLabel
