package rtb

func buildCreativeCache(reg *CampaignAuctionRegistry, creatives []CreativeData) {
	if reg == nil {
		return
	}
	resetCreativeCacheSoA(&reg.CreativeCache)
	count := reg.Count
	reg.CampaignCreativeStart = make([]uint32, count+1)
	if count == 0 || len(creatives) == 0 {
		return
	}

	campaignPos := make(map[CampaignID]int, count)
	for i := range count {
		campaignPos[reg.CampaignIDs[i]] = i
	}

	perCampaign := make([]uint32, count)
	prepared := make([]preparedCreative, 0, len(creatives))
	for i := range creatives {
		c := &creatives[i]
		catalogIdx, ok := campaignPos[c.CampaignID]
		if !ok {
			continue
		}
		wire, dur, err := PrepareCreativeVASTWire(c)
		if err != nil {
			continue
		}
		bid := c.Bid
		if bid == 0 {
			bid = reg.Bids[catalogIdx]
		}
		ctr := c.CTRPPM
		if ctr == 0 {
			ctr = reg.CTRPPM[catalogIdx]
		}
		weight := c.Weight
		if weight == 0 {
			weight = reg.Weights[catalogIdx]
		}
		media := uint8(c.MediaType)
		if media == 0 {
			media = uint8(MediaTypeDisplay)
		}
		perCampaign[catalogIdx]++
		prepared = append(prepared, preparedCreative{
			catalogIdx: uint32(catalogIdx),
			id:         c.ID,
			bid:        bid,
			ctr:        normalizeCTRPPM(ctr),
			weight:     weight,
			media:      media,
			duration:   dur,
			wire:       wire,
		})
	}

	total := 0
	for i := range count {
		reg.CampaignCreativeStart[i] = uint32(total)
		total += int(perCampaign[i])
	}
	reg.CampaignCreativeStart[count] = uint32(total)
	if total == 0 {
		return
	}

	soa := &reg.CreativeCache
	soa.CreativeIDs = make([]CreativeID, total)
	soa.CampaignIdx = make([]uint32, total)
	soa.Bids = make([]int64, total)
	soa.CTRPPM = make([]uint32, total)
	soa.Weights = make([]uint32, total)
	soa.MediaTypes = make([]uint8, total)
	soa.DurationSec = make([]uint32, total)
	soa.VASTWire = make([][]byte, total)

	writePos := make([]uint32, count)
	for i := range prepared {
		p := &prepared[i]
		ci := int(p.catalogIdx)
		slot := int(reg.CampaignCreativeStart[ci]) + int(writePos[ci])
		writePos[ci]++

		soa.CreativeIDs[slot] = p.id
		soa.CampaignIdx[slot] = p.catalogIdx
		soa.Bids[slot] = p.bid
		soa.CTRPPM[slot] = p.ctr
		soa.Weights[slot] = p.weight
		soa.MediaTypes[slot] = p.media
		soa.DurationSec[slot] = p.duration
		if len(p.wire) > 0 {
			owned := make([]byte, len(p.wire))
			copy(owned, p.wire)
			soa.VASTWire[slot] = owned
		}
	}
	soa.Count = total
}

type preparedCreative struct {
	catalogIdx uint32
	id         CreativeID
	bid        int64
	ctr        uint32
	weight     uint32
	media      uint8
	duration   uint32
	wire       []byte
}

func campaignCreativeRange(reg *CampaignAuctionRegistry, catalogIdx int) (start int, end int, ok bool) {
	if reg == nil || catalogIdx < 0 || catalogIdx >= reg.Count {
		return 0, 0, false
	}
	if len(reg.CampaignCreativeStart) != reg.Count+1 {
		return 0, 0, false
	}
	start = int(reg.CampaignCreativeStart[catalogIdx])
	end = int(reg.CampaignCreativeStart[catalogIdx+1])
	return start, end, end > start
}
