// Package outboxpb holds protobuf-generated wire types for outbox event payloads (api/outbox.proto).
//
// Role:
//   - CampaignPayload, SettingsPayload, and related messages serialized in outbox_events.payload JSON/binary.
//   - Consumed by internal/outbox appliers and controlplane outbox bridge type aliases.
//
// Invariants:
//   - Do not hand-edit outbox.pb.go; regenerate with make proto.
//
// Verify:
//
//	make proto
package outboxpb
