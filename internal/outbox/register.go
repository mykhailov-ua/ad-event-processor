package outbox

import (
	"time"

	"ad-event-processor/internal/controlplane/outboxpb"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

func init() {
	registerOutboxCodecs()
}

func registerOutboxCodecs() {
	coldpath.RegisterOutboxCodec(encodeCampaignPayload, decodeCampaignPayload)
	coldpath.RegisterOutboxCodec(encodeCampaignIDPayload, decodeCampaignIDPayload)
	coldpath.RegisterOutboxCodec(encodeBrandIDPayload, decodeBrandIDPayload)
	coldpath.RegisterOutboxCodec(encodeBrandFcapPayload, decodeBrandFcapPayload)
	coldpath.RegisterOutboxCodec(encodeCampaignSchedulePayload, decodeCampaignSchedulePayload)
	coldpath.RegisterOutboxCodec(encodeCampaignPacingPayload, decodeCampaignPacingPayload)
	coldpath.RegisterOutboxCodec(encodeSettingsPayload, decodeSettingsPayload)
	coldpath.RegisterOutboxCodec(encodeBlacklistPayload, decodeBlacklistPayload)
	coldpath.RegisterOutboxCodec(encodeFraudThreatPayload, decodeFraudThreatPayload)
	coldpath.RegisterOutboxCodec(encodeFraudModelVersionPayload, decodeFraudModelVersionPayload)
	coldpath.RegisterOutboxCodec(encodeUserConsentPayload, decodeUserConsentPayload)
	coldpath.RegisterOutboxCodec(encodePurgeUserDataPayload, decodePurgeUserDataPayload)
	coldpath.RegisterOutboxCodec(encodePausePlacementPayload, decodePausePlacementPayload)
	coldpath.RegisterOutboxCodec(encodeQuotaRepairPayload, decodeQuotaRepairPayload)
	coldpath.RegisterOutboxCodec(encodeReconciliationAdjustPayload, decodeReconciliationAdjustPayload)
	coldpath.RegisterOutboxCodec(encodeSupplyFilesPayload, decodeSupplyFilesPayload)
	coldpath.RegisterOutboxCodec(encodeRtbCatalogReloadPayload, decodeRtbCatalogReloadPayload)
	coldpath.RegisterOutboxCodec(encodeCohortSnapshotPayload, decodeCohortSnapshotPayload)
	coldpath.RegisterOutboxCodec(encodeCtvGtaxSettlementPayload, decodeCtvGtaxSettlementPayload)
	coldpath.RegisterOutboxCodec(encodeTelegramEventPayload, decodeTelegramEventPayload)
}

func encodeCampaignPayload(p CampaignPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.CampaignPayload{
		CampaignId:  p.CampaignID,
		BudgetLimit: p.BudgetLimit,
	})
}

func decodeCampaignPayload(b []byte) (CampaignPayload, error) {
	var pb outboxpb.CampaignPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return CampaignPayload{}, err
	}
	return CampaignPayload{CampaignID: pb.CampaignId, BudgetLimit: pb.BudgetLimit}, nil
}

func encodeCampaignIDPayload(p CampaignIDPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.CampaignIdPayload{CampaignId: p.CampaignID})
}

func decodeCampaignIDPayload(b []byte) (CampaignIDPayload, error) {
	var pb outboxpb.CampaignIdPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return CampaignIDPayload{}, err
	}
	return CampaignIDPayload{CampaignID: pb.CampaignId}, nil
}

func encodeBrandIDPayload(p BrandIDPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.BrandIdPayload{BrandId: p.BrandID})
}

func decodeBrandIDPayload(b []byte) (BrandIDPayload, error) {
	var pb outboxpb.BrandIdPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return BrandIDPayload{}, err
	}
	return BrandIDPayload{BrandID: pb.BrandId}, nil
}

func encodeBrandFcapPayload(p BrandFcapPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.BrandFcapPayload{
		BrandId:    p.BrandID,
		FreqLimit:  p.FreqLimit,
		FreqWindow: p.FreqWindow,
	})
}

func decodeBrandFcapPayload(b []byte) (BrandFcapPayload, error) {
	var pb outboxpb.BrandFcapPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return BrandFcapPayload{}, err
	}
	return BrandFcapPayload{
		BrandID:    pb.BrandId,
		FreqLimit:  pb.FreqLimit,
		FreqWindow: pb.FreqWindow,
	}, nil
}

func encodeCampaignSchedulePayload(p CampaignSchedulePayload) ([]byte, error) {
	msg := &outboxpb.CampaignSchedulePayload{
		CampaignId: p.CampaignID,
	}
	if len(p.DaypartHours) > 0 {
		msg.DaypartHours = make([]int32, len(p.DaypartHours))
		for i, h := range p.DaypartHours {
			msg.DaypartHours[i] = int32(h)
		}
	}
	if p.StartAt != nil {
		msg.StartAtUnix = p.StartAt.Unix()
	}
	if p.EndAt != nil {
		msg.EndAtUnix = p.EndAt.Unix()
	}
	return proto.Marshal(msg)
}

func decodeCampaignSchedulePayload(b []byte) (CampaignSchedulePayload, error) {
	var pb outboxpb.CampaignSchedulePayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return CampaignSchedulePayload{}, err
	}
	out := CampaignSchedulePayload{
		CampaignID:   pb.CampaignId,
		DaypartHours: append([]int16(nil), int16SliceFromInt32(pb.DaypartHours)...),
	}
	if pb.StartAtUnix != 0 {
		t := time.Unix(pb.StartAtUnix, 0).UTC()
		out.StartAt = &t
	}
	if pb.EndAtUnix != 0 {
		t := time.Unix(pb.EndAtUnix, 0).UTC()
		out.EndAt = &t
	}
	return out, nil
}

func encodeCampaignPacingPayload(p CampaignPacingPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.CampaignPacingPayload{
		CampaignId: p.CampaignID,
		PacingMode: p.PacingMode,
	})
}

func decodeCampaignPacingPayload(b []byte) (CampaignPacingPayload, error) {
	var pb outboxpb.CampaignPacingPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return CampaignPacingPayload{}, err
	}
	return CampaignPacingPayload{CampaignID: pb.CampaignId, PacingMode: pb.PacingMode}, nil
}

func encodeSettingsPayload(p SettingsPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.SettingsPayload{Settings: p.Settings})
}

func decodeSettingsPayload(b []byte) (SettingsPayload, error) {
	var pb outboxpb.SettingsPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return SettingsPayload{}, err
	}
	return SettingsPayload{Settings: pb.Settings}, nil
}

func encodeBlacklistPayload(p BlacklistPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.BlacklistPayload{
		Action: p.Action,
		Ip:     p.IP,
		Reason: p.Reason,
	})
}

func decodeBlacklistPayload(b []byte) (BlacklistPayload, error) {
	var pb outboxpb.BlacklistPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return BlacklistPayload{}, err
	}
	return BlacklistPayload{Action: pb.Action, IP: pb.Ip, Reason: pb.Reason}, nil
}

func encodeFraudThreatPayload(p FraudThreatPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.FraudThreatPayload{
		Action:     p.Action,
		Ip:         p.IP,
		CampaignId: p.CampaignID,
		Score:      p.Score,
		Boost:      p.Boost,
		TtlSeconds: p.TTLSeconds,
	})
}

func decodeFraudThreatPayload(b []byte) (FraudThreatPayload, error) {
	var pb outboxpb.FraudThreatPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return FraudThreatPayload{}, err
	}
	return FraudThreatPayload{
		Action:     pb.Action,
		IP:         pb.Ip,
		CampaignID: pb.CampaignId,
		Score:      pb.Score,
		Boost:      pb.Boost,
		TTLSeconds: pb.TtlSeconds,
	}, nil
}

func encodeFraudModelVersionPayload(p FraudModelVersionPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.FraudModelVersionPayload{
		ModelVersion: p.ModelVersion,
		Hash:         p.Hash,
		ShardId:      int32(p.ShardID),
	})
}

func decodeFraudModelVersionPayload(b []byte) (FraudModelVersionPayload, error) {
	var pb outboxpb.FraudModelVersionPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return FraudModelVersionPayload{}, err
	}
	return FraudModelVersionPayload{
		ModelVersion: pb.ModelVersion,
		Hash:         pb.Hash,
		ShardID:      int(pb.ShardId),
	}, nil
}

func encodeUserConsentPayload(p UserConsentPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.UserConsentPayload{
		UserIdHash: p.UserIDHash,
		Purposes:   int32(p.Purposes),
	})
}

func decodeUserConsentPayload(b []byte) (UserConsentPayload, error) {
	var pb outboxpb.UserConsentPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return UserConsentPayload{}, err
	}
	return UserConsentPayload{UserIDHash: pb.UserIdHash, Purposes: int16(pb.Purposes)}, nil
}

func encodePurgeUserDataPayload(p PurgeUserDataPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.PurgeUserDataPayload{
		ErasureId:     p.ErasureID,
		UserIdHash:    p.UserIDHash,
		SubjectUserId: p.SubjectUserID,
	})
}

func decodePurgeUserDataPayload(b []byte) (PurgeUserDataPayload, error) {
	var pb outboxpb.PurgeUserDataPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return PurgeUserDataPayload{}, err
	}
	return PurgeUserDataPayload{
		ErasureID:     pb.ErasureId,
		UserIDHash:    pb.UserIdHash,
		SubjectUserID: pb.SubjectUserId,
	}, nil
}

func encodePausePlacementPayload(p PausePlacementPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.PausePlacementPayload{
		CampaignId:  p.CampaignID,
		PlacementId: p.PlacementID,
		Action:      p.Action,
	})
}

func decodePausePlacementPayload(b []byte) (PausePlacementPayload, error) {
	var pb outboxpb.PausePlacementPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return PausePlacementPayload{}, err
	}
	return PausePlacementPayload{
		CampaignID:  pb.CampaignId,
		PlacementID: pb.PlacementId,
		Action:      pb.Action,
	}, nil
}

func encodeQuotaRepairPayload(p QuotaRepairPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.QuotaRepairPayload{
		CampaignId:       p.CampaignID,
		ShardId:          int32(p.ShardID),
		Action:           p.Action,
		PostgresReserved: p.PostgresReserved,
		RedisExpected:    p.RedisExpected,
		ChunkSize:        p.ChunkSize,
		DriftMicro:       p.DriftMicro,
		RepairMicro:      p.RepairMicro,
		Reason:           p.Reason,
	})
}

func decodeQuotaRepairPayload(b []byte) (QuotaRepairPayload, error) {
	var pb outboxpb.QuotaRepairPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return QuotaRepairPayload{}, err
	}
	return QuotaRepairPayload{
		CampaignID:       pb.CampaignId,
		ShardID:          int16(pb.ShardId),
		Action:           pb.Action,
		PostgresReserved: pb.PostgresReserved,
		RedisExpected:    pb.RedisExpected,
		ChunkSize:        pb.ChunkSize,
		DriftMicro:       pb.DriftMicro,
		RepairMicro:      pb.RepairMicro,
		Reason:           pb.Reason,
	}, nil
}

func encodeReconciliationAdjustPayload(p ReconciliationAdjustPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.ReconciliationAdjustPayload{
		RunId:             p.RunID,
		CampaignId:        p.CampaignID,
		CustomerId:        p.CustomerID,
		ShardId:           int32(p.ShardID),
		LedgerAmountMicro: p.LedgerAmt,
		RedisDeltaMicro:   p.RedisDelta,
		Reason:            p.Reason,
	})
}

func decodeReconciliationAdjustPayload(b []byte) (ReconciliationAdjustPayload, error) {
	var pb outboxpb.ReconciliationAdjustPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return ReconciliationAdjustPayload{}, err
	}
	return ReconciliationAdjustPayload{
		RunID:      pb.RunId,
		CampaignID: pb.CampaignId,
		CustomerID: pb.CustomerId,
		ShardID:    int16(pb.ShardId),
		LedgerAmt:  pb.LedgerAmountMicro,
		RedisDelta: pb.RedisDeltaMicro,
		Reason:     pb.Reason,
	}, nil
}

func encodeSupplyFilesPayload(p SupplyFilesPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.SupplyFilesPayload{Trigger: p.Trigger})
}

func decodeSupplyFilesPayload(b []byte) (SupplyFilesPayload, error) {
	var pb outboxpb.SupplyFilesPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return SupplyFilesPayload{}, err
	}
	return SupplyFilesPayload{Trigger: pb.Trigger}, nil
}

func encodeRtbCatalogReloadPayload(p RtbCatalogReloadPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.RtbCatalogReloadPayload{Trigger: p.Trigger})
}

func decodeRtbCatalogReloadPayload(b []byte) (RtbCatalogReloadPayload, error) {
	var pb outboxpb.RtbCatalogReloadPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return RtbCatalogReloadPayload{}, err
	}
	return RtbCatalogReloadPayload{Trigger: pb.Trigger}, nil
}

func encodeCohortSnapshotPayload(p CohortSnapshotPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.CohortSnapshotPayload{Version: int32(p.Version)})
}

func decodeCohortSnapshotPayload(b []byte) (CohortSnapshotPayload, error) {
	var pb outboxpb.CohortSnapshotPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return CohortSnapshotPayload{}, err
	}
	return CohortSnapshotPayload{Version: int64(pb.Version)}, nil
}

func encodeCtvGtaxSettlementPayload(p CTVGtaxSettlementPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.CtvGtaxSettlementPayload{
		SettlementId: p.SettlementID,
		CustomerId:   p.CustomerID,
		CampaignId:   p.CampaignID,
		SpendMicro:   p.SpendMicro,
	})
}

func decodeCtvGtaxSettlementPayload(b []byte) (CTVGtaxSettlementPayload, error) {
	var pb outboxpb.CtvGtaxSettlementPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return CTVGtaxSettlementPayload{}, err
	}
	return CTVGtaxSettlementPayload{
		SettlementID: pb.SettlementId,
		CustomerID:   pb.CustomerId,
		CampaignID:   pb.CampaignId,
		SpendMicro:   pb.SpendMicro,
	}, nil
}

func encodeTelegramEventPayload(p TelegramEventPayload) ([]byte, error) {
	return proto.Marshal(&outboxpb.TelegramEventPayload{
		CampaignId: p.CampaignID[:],
		BotId:      p.BotID,
		Payload:    append([]byte(nil), p.Payload...),
	})
}

func decodeTelegramEventPayload(b []byte) (TelegramEventPayload, error) {
	var pb outboxpb.TelegramEventPayload
	if err := coldpath.UnmarshalProtoMessage(b, &pb); err != nil {
		return TelegramEventPayload{}, err
	}
	var campID uuid.UUID
	if len(pb.CampaignId) == 16 {
		copy(campID[:], pb.CampaignId)
	}
	return TelegramEventPayload{
		CampaignID: campID,
		BotID:      pb.BotId,
		Payload:    append([]byte(nil), pb.Payload...),
	}, nil
}

func int16SliceFromInt32(in []int32) []int16 {
	if len(in) == 0 {
		return nil
	}
	out := make([]int16, len(in))
	for i, v := range in {
		out[i] = int16(v)
	}
	return out
}
