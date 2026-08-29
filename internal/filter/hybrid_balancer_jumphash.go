//go:build jumphash

package filter

import (
	"hash/fnv"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

var randSeedSeq atomic.Int64

var randPool = sync.Pool{
	New: func() any {
		seed := time.Now().UnixNano() ^ randSeedSeq.Add(1)
		return rand.New(rand.NewSource(seed))
	},
}

func jumpHash(key uint64, numBuckets int32) int32 {
	var b int64 = -1
	var j int64
	for j < int64(numBuckets) {
		b = j
		key = key*2862933555777941757 + 1
		j = int64(float64(b+1) * (float64(1<<31) / float64((key>>33)+1)))
	}
	return int32(b)
}

func (hb *HybridBalancer) SelectAndShard(userID string, currentCampaignRps int64) (*CampaignMeta, int) {
	table := hb.aliasTable.Load()
	if table == nil || len(table.prob) == 0 {
		return nil, 0
	}

	n := len(table.prob)
	r := randPool.Get().(*rand.Rand)
	idx := r.Intn(n)

	selectedIdx := idx
	if r.Float64() >= table.prob[idx] {
		selectedIdx = table.alias[idx]
	}
	randPool.Put(r)

	campaign := table.campaigns[selectedIdx]
	if hb.totalShards <= 0 {
		return campaign, 0
	}

	isHot := hb.maxRpsPerNode > 0 && currentCampaignRps > hb.maxRpsPerNode
	var shard int

	if !isHot {
		shard = int(jumpHash(uint64(CRC32Castagnoli(&campaign.ID)), int32(hb.totalShards)))
	} else {
		subShardCount := int(currentCampaignRps/hb.maxRpsPerNode) + 1
		if subShardCount > hb.totalShards {
			subShardCount = hb.totalShards
		}
		if subShardCount <= 0 {
			subShardCount = 1
		}

		hasher := fnv.New32a()
		_, _ = hasher.Write([]byte(userID))
		userHash := hasher.Sum32()
		subShardIdx := userHash % uint32(subShardCount)

		combinedHash := uint64(CRC32Castagnoli(&campaign.ID)) ^ uint64(subShardIdx)
		shard = int(jumpHash(combinedHash, int32(hb.totalShards)))
	}

	return campaign, shard
}
