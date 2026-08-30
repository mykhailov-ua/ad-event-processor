package ingest

import (
	"ad-event-processor/internal/domain/shard"
	filterunified "ad-event-processor/internal/filter/unified"
)

const udpProtocolVersion2 = shard.UDPProtocolVersion2

func udpEncodeShardLimits(dst []byte, limits *UDPControlLimits) int {
	return shard.UDPEncodeShardLimits(dst, limits)
}

func udpDecodeShardLimits(payload []byte, numShards uint8, version uint8, out *UDPControlLimits) bool {
	return shard.UDPDecodeShardLimits(payload, numShards, version, out)
}

func udpEncodeNodeWeights(dst []byte, weights []UDPNodeWeight) int {
	return shard.UDPEncodeNodeWeights(dst, weights)
}

func udpDecodeNodeWeights(payload []byte, out *[]UDPNodeWeight) bool {
	return shard.UDPDecodeNodeWeights(payload, out)
}

var parseRedisUsedMemory = filterunified.ParseRedisUsedMemory
