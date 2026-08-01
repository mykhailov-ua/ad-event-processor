package ingestion

import (
	"encoding/binary"
	"hash/crc32"
	"unsafe"

	"espx/internal/domain"
	"espx/internal/rtb"

	"github.com/google/uuid"
)

type BudgetAuthority uint8

const (
	BudgetAuthorityRedis BudgetAuthority = iota
	BudgetAuthorityRTB
	BudgetAuthorityShadow
)

type RtbCampaignInput struct {
	BidMicro         int64
	CTRPPM           uint32
	ReserveMicro     int64
	DailyBudgetMicro int64
	PacingOpen       uint8
	CustomerID       rtb.CustomerID
	CustomerBudget   int64
	DeviceMask       uint8
	CategoryMask     uint64
	GeoHash          uint32
	Weight           uint32
	BoostPPM         uint32
}

type RtbTargetingInput struct {
	GeoHash             uint32
	DeviceType          uint8
	CategoryMask        uint64
	PublisherFloorMicro int64
	DealIDLen           uint8
	DealIDBuf           [64]byte
	SeatCount           uint8
	DeadlineMono        int64
	DealBlock           rtb.NoBidReason
	Schain              SchainNodes
	SchainCount         uint8
	FcapUserHash        uint64
	ConnectionType      uint8
	ViewabilityPPM      uint32
	PMPPrivate          uint8
	DeviceLMT           uint8
	BlockedCatMask      uint64
}

func CampaignIDFromUUID(id uuid.UUID) rtb.CampaignID {
	return rtb.CampaignID(binary.BigEndian.Uint64(id[:8]))
}

func GeoHashFromCountry(country string) uint32 {
	if country == "" {
		return 0
	}
	return crc32.ChecksumIEEE([]byte(country))
}

func GeoHashFromCountryBytes(country []byte) uint32 {
	if len(country) == 0 {
		return 0
	}
	return crc32.ChecksumIEEE(country)
}

func DeviceMaskFromType(deviceType []byte) uint8 {
	switch len(deviceType) {
	case 6:
		if deviceType[0] == 'm' && deviceType[1] == 'o' && deviceType[2] == 'b' &&
			deviceType[3] == 'i' && deviceType[4] == 'l' && deviceType[5] == 'e' {
			return 2
		}
		if deviceType[0] == 't' && deviceType[1] == 'a' && deviceType[2] == 'b' &&
			deviceType[3] == 'l' && deviceType[4] == 'e' && deviceType[5] == 't' {
			return 4
		}
	case 7:
		if deviceType[0] == 'd' && deviceType[1] == 'e' && deviceType[2] == 's' &&
			deviceType[3] == 'k' && deviceType[4] == 't' && deviceType[5] == 'o' &&
			deviceType[6] == 'p' {
			return 1
		}
	}
	return 1
}

func BidRequestFromEvent(evt *domain.Event, targeting RtbTargetingInput) rtb.BidRequest {
	fcapUserHash := targeting.FcapUserHash
	if fcapUserHash == 0 && evt != nil && evt.UserID != "" {
		fcapUserHash = hashUserID(evt.UserID)
	}
	return rtb.BidRequest{
		CategoryMask:   targeting.CategoryMask,
		MinBid:         targeting.PublisherFloorMicro,
		GeoHash:        targeting.GeoHash,
		DeviceType:     targeting.DeviceType,
		DeadlineMono:   targeting.DeadlineMono,
		DealBlock:      targeting.DealBlock,
		NowUnix:        CachedTimeUTC().Unix(),
		FcapUserHash:   fcapUserHash,
		BlockedCatMask: targeting.BlockedCatMask,
	}
}

func hashUserID(userID string) uint64 {
	if userID == "" {
		return 0
	}
	return rtb.HashBytes64(unsafe.Slice(unsafe.StringData(userID), len(userID)))
}

func hashUserIDBytes(userID []byte) uint64 {
	if len(userID) == 0 {
		return 0
	}
	return rtb.HashBytes64(userID)
}

func CampaignDataFromDomain(camp *domain.Campaign, input RtbCampaignInput) rtb.CampaignData {
	remaining := camp.BudgetLimit - camp.CurrentSpend
	if remaining < 0 {
		remaining = 0
	}
	daypartMask, tzOffset, startUnix, endUnix := scheduleFieldsFromCampaign(camp)
	var freqLimit uint32
	if camp.FreqLimit > 0 {
		freqLimit = uint32(camp.FreqLimit)
	}
	var fcapPrefixHash uint64
	if camp.FcapKeyPrefix != "" {
		fcapPrefixHash = rtb.HashBytes64([]byte(camp.FcapKeyPrefix))
	}
	return rtb.CampaignData{
		ID:             CampaignIDFromUUID(camp.ID),
		Bid:            input.BidMicro,
		CTRPPM:         input.CTRPPM,
		Reserve:        input.ReserveMicro,
		DailyBudget:    input.DailyBudgetMicro,
		PacingOpen:     input.PacingOpen,
		CustomerID:     input.CustomerID,
		CustomerBudget: input.CustomerBudget,
		DeviceMask:     input.DeviceMask,
		CategoryMask:   input.CategoryMask,
		GeoHashVal:     input.GeoHash,
		Weight:         input.Weight,
		BoostPPM:       input.BoostPPM,
		Budget:         remaining,
		DaypartMask:    daypartMask,
		TZOffsetSec:    tzOffset,
		ScheduleStart:  startUnix,
		ScheduleEnd:    endUnix,
		FreqLimit:      freqLimit,
		FcapPrefixHash: fcapPrefixHash,
	}
}

func scheduleFieldsFromCampaign(camp *domain.Campaign) (mask uint32, tzOffset int32, startUnix, endUnix int64) {
	if camp == nil {
		return 0, 0, 0, 0
	}
	mask = rtb.DaypartMaskFromHours(camp.DaypartHours)
	now := CachedTimeUTC()
	if camp.Location != nil {
		_, off := now.In(camp.Location).Zone()
		tzOffset = int32(off)
	}
	if camp.StartAt != nil {
		startUnix = camp.StartAt.Unix()
	}
	if camp.EndAt != nil {
		endUnix = camp.EndAt.Unix()
	}
	return mask, tzOffset, startUnix, endUnix
}

func BuildRtbCatalogRows(campaigns []*domain.Campaign, inputs map[uuid.UUID]RtbCampaignInput) []rtb.CampaignData {
	if len(campaigns) == 0 {
		return nil
	}
	out := make([]rtb.CampaignData, 0, len(campaigns))
	for _, camp := range campaigns {
		if camp == nil {
			continue
		}
		input, ok := inputs[camp.ID]
		if !ok {
			continue
		}
		out = append(out, fanOutRtbCatalogRows(camp, input)...)
	}
	return out
}
