// Package outboxpb holds protobuf-generated wire types for outbox event payloads.
//
// Role:
//   - Generated from api/outbox.proto (protobuf package coldpath.v1).
//   - Binary payload bodies stored in outbox_events.payload after domain handlers enqueue.
//   - internal/outbox/register.go registers proto.Marshal/Unmarshal codecs via pkg/coldpath.RegisterOutboxCodec.
//   - Domain packages enqueue typed payloads; OutboxWorker appliers decode through the registered codecs.
//
// Wired payload messages (codecs in internal/outbox/register.go):
//   - CampaignPayload, CampaignIdPayload, BrandIdPayload, BrandFcapPayload
//   - CampaignSchedulePayload, CampaignPacingPayload, SettingsPayload
//   - BlacklistPayload, FraudThreatPayload, FraudModelVersionPayload
//   - UserConsentPayload, PurgeUserDataPayload, PausePlacementPayload
//   - QuotaRepairPayload, ReconciliationAdjustPayload
//   - SupplyFilesPayload, RtbCatalogReloadPayload, CohortSnapshotPayload
//   - CtvGtaxSettlementPayload, TelegramEventPayload
//
// Proto-only (no outbox codec registered yet):
//   - SettleBalancePayload, ChargebackPayload — payment settlement still JSON-encodes in internal/payment/settlement.
//
// Invariants:
//   - Do not hand-edit outbox.pb.go; regenerate with make proto after api/outbox.proto changes.
//   - Financial amounts in proto fields are micro-units (10^-6 of base currency) where documented in .proto.
//   - New outbox event types: extend api/outbox.proto, make proto, then add codec pair in internal/outbox/register.go.
//
// Forbidden:
//   - Business logic or outbox polling in this package (generated types only).
//   - Tracker hot-path imports.
//
// Verify:
//   go list -e ./internal/controlplane/outboxpb/
//   make proto
//   go test ./internal/outbox/ -short -run TestProto_roundTripCampaign -count=1
//   go test ./internal/outbox/ -short -run TestProto_legacyJSONCampaign -count=1
package outboxpb
