package cold

import (
	"hash/crc32"

	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/filter/netintel"
	"ad-event-processor/internal/ingest/pb"

	"github.com/google/uuid"
)

func FormatUUIDCanonical(dst *[36]byte, id uuid.UUID) {
	b := dst[:]
	b[0] = filter.HexChars[id[0]>>4]
	b[1] = filter.HexChars[id[0]&0xf]
	b[2] = filter.HexChars[id[1]>>4]
	b[3] = filter.HexChars[id[1]&0xf]
	b[4] = filter.HexChars[id[2]>>4]
	b[5] = filter.HexChars[id[2]&0xf]
	b[6] = filter.HexChars[id[3]>>4]
	b[7] = filter.HexChars[id[3]&0xf]
	b[8] = '-'
	b[9] = filter.HexChars[id[4]>>4]
	b[10] = filter.HexChars[id[4]&0xf]
	b[11] = filter.HexChars[id[5]>>4]
	b[12] = filter.HexChars[id[5]&0xf]
	b[13] = '-'
	b[14] = filter.HexChars[id[6]>>4]
	b[15] = filter.HexChars[id[6]&0xf]
	b[16] = filter.HexChars[id[7]>>4]
	b[17] = filter.HexChars[id[7]&0xf]
	b[18] = '-'
	b[19] = filter.HexChars[id[8]>>4]
	b[20] = filter.HexChars[id[8]&0xf]
	b[21] = filter.HexChars[id[9]>>4]
	b[22] = filter.HexChars[id[9]&0xf]
	b[23] = '-'
	b[24] = filter.HexChars[id[10]>>4]
	b[25] = filter.HexChars[id[10]&0xf]
	b[26] = filter.HexChars[id[11]>>4]
	b[27] = filter.HexChars[id[11]&0xf]
	b[28] = filter.HexChars[id[12]>>4]
	b[29] = filter.HexChars[id[12]&0xf]
	b[30] = filter.HexChars[id[13]>>4]
	b[31] = filter.HexChars[id[13]&0xf]
	b[32] = filter.HexChars[id[14]>>4]
	b[33] = filter.HexChars[id[14]&0xf]
	b[34] = filter.HexChars[id[15]>>4]
	b[35] = filter.HexChars[id[15]&0xf]
}

func ComputeCompositeHashUUID(campaignID uuid.UUID, userID []byte) uint32 {
	var crc uint32
	var started bool

	if campaignID != uuid.Nil {
		var buf [36]byte
		FormatUUIDCanonical(&buf, campaignID)
		crc = crc32IEEEInplace36(&buf)
		started = true
	}
	if len(userID) > 0 {
		if started {
			crc = crc32.Update(crc, crc32.IEEETable, userID)
		} else {
			crc = crc32.ChecksumIEEE(userID)
		}
	}
	if !started && len(userID) == 0 {
		return 0
	}
	return crc
}

func crc32IEEEInplace36(b *[36]byte) uint32 {
	crc := ^uint32(0)
	tab := crc32.IEEETable
	for i := range 36 {
		crc = tab[byte(crc)^b[i]] ^ crc>>8
	}
	return ^crc
}

func ComputeCompositeHashFromTrackCampaignUser(campaignID uuid.UUID, userID string) uint32 {
	return ComputeCompositeHashUUID(campaignID, filter.UnsafeBytes(userID))
}

func ComputeCompositeHashFromProto(req *pb.AdEvent) uint32 {
	var camp uuid.UUID
	if len(req.CampaignId) == 16 {
		copy(camp[:], req.CampaignId)
	}
	var userID []byte
	if req.Metadata != nil {
		userID = req.Metadata.UserId
	}
	return ComputeCompositeHashUUID(camp, userID)
}

func ResetAdEventInPlace(evt *pb.AdEvent) {
	evt.CampaignId = evt.CampaignId[:0]
	evt.EventType = evt.EventType[:0]
	if evt.Metadata != nil {
		evt.Metadata.ClickId = evt.Metadata.ClickId[:0]
		evt.Metadata.UserId = evt.Metadata.UserId[:0]
		evt.Metadata.DeviceType = evt.Metadata.DeviceType[:0]
		evt.Metadata.Os = evt.Metadata.Os[:0]
		for i := range evt.Metadata.ExtraKeys {
			evt.Metadata.ExtraKeys[i] = evt.Metadata.ExtraKeys[i][:0]
		}
		evt.Metadata.ExtraKeys = evt.Metadata.ExtraKeys[:0]
		for i := range evt.Metadata.ExtraValues {
			evt.Metadata.ExtraValues[i] = evt.Metadata.ExtraValues[i][:0]
		}
		evt.Metadata.ExtraValues = evt.Metadata.ExtraValues[:0]
		evt.Metadata.ExtraBytes = evt.Metadata.ExtraBytes[:0]
	}
}

type ConversionDatacenterIPChecker struct {
	geo netintel.GeoProvider
	dc  *netintel.DCASNTable
}

func NewConversionDatacenterIPChecker(geo netintel.GeoProvider, dc *netintel.DCASNTable) *ConversionDatacenterIPChecker {
	if geo == nil && dc == nil {
		return nil
	}
	return &ConversionDatacenterIPChecker{geo: geo, dc: dc}
}

func (c *ConversionDatacenterIPChecker) IsDatacenterIP(ip string) bool {
	if c == nil || ip == "" {
		return false
	}
	if c.geo != nil {
		anon, err := c.geo.IsAnonymous(ip)
		if err == nil && anon {
			return true
		}
	}
	if c.dc == nil || !c.dc.Ready() || c.geo == nil {
		return false
	}
	lookup, ok := c.geo.(filter.ASNLookup)
	if !ok {
		return false
	}
	asn, asnOK := lookup.LookupASN(ip)
	if !asnOK || asn == 0 {
		return false
	}
	return c.dc.IsDatacenter(asn)
}
