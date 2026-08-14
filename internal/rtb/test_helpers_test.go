package rtb

func itoaU64(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

func singleCampaign(id CampaignID, bid int64, budget int64) []CampaignData {
	return []CampaignData{{
		ID:           id,
		Bid:          bid,
		DeviceMask:   1,
		CategoryMask: 1,
		GeoHashVal:   7,
		Weight:       1,
		Budget:       budget,
	}}
}

func stdReq(geo uint32, minBid int64) *BidRequest {
	return &BidRequest{
		DeviceType:   1,
		CategoryMask: 1,
		GeoHash:      geo,
		MinBid:       minBid,
	}
}

func catalogIDs(reg *Registry) map[CampaignID]struct{} {
	out := make(map[CampaignID]struct{})
	snap := reg.loadCatalog()
	if snap == nil {
		return out
	}
	for i := range geoShardCount {
		sh := snap.shards[i]
		if sh == nil || sh.Count == 0 {
			continue
		}
		for j := 0; j < sh.Count && j < len(sh.CampaignIDs); j++ {
			out[sh.CampaignIDs[j]] = struct{}{}
		}
	}
	return out
}
