package rtb

func (registry *Registry) LookupCreativeWire(geoHash uint32, campaignID CampaignID, creativeID CreativeID) ([]byte, uint8, bool) {
	reg := registry.LoadShard(geoHash)
	if reg == nil || reg.Count == 0 {
		return nil, 0, false
	}
	catalogIdx := -1
	for i := 0; i < reg.Count; i++ {
		if reg.CampaignIDs[i] == campaignID {
			catalogIdx = i
			break
		}
	}
	if catalogIdx < 0 {
		return nil, 0, false
	}
	start, end, ok := campaignCreativeRange(reg, catalogIdx)
	if !ok {
		return nil, 0, false
	}
	soa := &reg.CreativeCache
	for slot := start; slot < end; slot++ {
		if soa.CreativeIDs[slot] == creativeID {
			wire := soa.VASTWire[slot]
			if len(wire) == 0 {
				return nil, soa.MediaTypes[slot], false
			}
			return wire, soa.MediaTypes[slot], true
		}
	}
	return nil, 0, false
}
