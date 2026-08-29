package rtb

import (
	"sort"
)

func (r *CampaignAuctionRegistry) geoRange(geoHash uint32) (start int, end int, ok bool) {
	if r == nil || r.GeoBucketCount == 0 {
		return 0, 0, false
	}
	hashes := r.GeoBucketHash
	idx := sort.Search(r.GeoBucketCount, func(i int) bool {
		return hashes[i] >= geoHash
	})
	if idx >= r.GeoBucketCount || hashes[idx] != geoHash {
		return 0, 0, false
	}
	start = int(r.GeoBucketStart[idx])
	end = int(r.GeoBucketStart[idx+1])
	return start, end, true
}

func buildGeoIndex(reg *CampaignAuctionRegistry) {
	if reg == nil || reg.Count == 0 {
		reg.GeoBucketCount = 0
		resetBucketSoA(&reg.GeoBucketSoA)
		return
	}

	buckets := make(map[uint32][]uint32, reg.Count)
	for i := range reg.Count {
		geo := reg.GeoHashes[i]
		buckets[geo] = append(buckets[geo], uint32(i))
	}

	reg.GeoBucketCount = len(buckets)
	reg.GeoBucketHash = make([]uint32, 0, reg.GeoBucketCount)
	for geo := range buckets {
		reg.GeoBucketHash = append(reg.GeoBucketHash, geo)
	}
	sort.Slice(reg.GeoBucketHash, func(i, j int) bool {
		return reg.GeoBucketHash[i] < reg.GeoBucketHash[j]
	})

	reg.GeoBucketStart = make([]uint32, reg.GeoBucketCount+1)
	resetBucketSoA(&reg.GeoBucketSoA)
	ensureBucketSoACap(&reg.GeoBucketSoA, reg.Count)
	for i, geo := range reg.GeoBucketHash {
		reg.GeoBucketStart[i] = uint32(reg.GeoBucketSoA.len())
		for _, catalogIdx := range buckets[geo] {
			appendBucketCandidate(&reg.GeoBucketSoA, reg, catalogIdx)
		}
	}
	reg.GeoBucketStart[reg.GeoBucketCount] = uint32(reg.GeoBucketSoA.len())
}
