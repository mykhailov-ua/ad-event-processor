// Package watchers runs background campaign registry and slot-map reload loops.
//
// Role:
//   - CampaignUpdateWatcher consumes the broker campaign-update topic and patches filter.Registry
//     (incremental UpdateAndWarmCampaign or full ReloadFullSnapshot on sync payload).
//   - SlotMapWatcher reloads StaticSlot shard table from Postgres on poll interval and on broker reload signals.
//   - Uses ingest/parser.ParseUUID for broker campaign-id payloads.
//
// Topology:
//   - Background goroutines started from cmd/tracker wire; not on synchronous /track or /click accept path.
//   - Broker consumer with exponential backoff (1s..30s); optional BrokerRedisURL for offset commits.
//   - SlotMapWatcher runs pollLoop and optional brokerLoop concurrently (sync.WaitGroup).
//   - Fetch batch size 64 KiB; 500 ms idle between broker fetch iterations.
//
// Thread model (hot-path.mdc Tracker thread model):
//
//	Off ingest Tier A/B request path entirely: watchers never run inside gnet OnTraffic or PinnedWorkerPool
//	  offload handlers. Hot readers observe Registry and StaticSlotSharder snapshots via atomic.Pointer
//	  swap after background reload completes.
//
// Invariants:
//   - Registry reload uses atomic.Pointer swap inside filter.Registry; Tier B readers never block on watcher I/O.
//   - Invalid broker payloads are logged and skipped; offsets still advance to avoid poison-pill stall.
//   - Campaign full sync: domain.IsRegistryFullSyncPayload triggers ReloadFullSnapshot; UUID payload triggers UpdateAndWarmCampaign.
//   - Slot map: shard.ReloadStaticSlotMapIfChanged publishes only when PG version changes; broker signal triggers tryReload.
//   - Default poll interval 10s when SlotMapWatcherConfig.PollInterval unset; startup tryReload before first tick.
//   - Broker group defaults to campaign-update-<hostname> or slotmap-<hostname> when unset.
//
// Contracts:
//   - Campaign topic default domain.DefaultCampaignUpdateBrokerTopic; slot map domain.DefaultSlotMapReloadTopic.
//   - Slot map broker payload decoded via shard.DecodeSlotMapReloadMessage (invalid decode skipped, offset advanced).
//   - MarkPubSubOK on successful registry incremental or full reload.
//   - CommitOffset stores next offset only when messages were consumed and nextOffset > start.
//
// Tradeoffs:
//   - Rejected synchronous broker or Postgres fetch inside FilterEngine.Check, processTrack, or Tier A gnet.
//   - Rejected blocking hot readers during reload: RCU atomic.Pointer swap after PG/broker work completes.
//   - Rejected poison-pill stall: skip invalid payload, commit offset, retain last good registry/slot map snapshot.
//   - Reload fail-open on error: failed tryReload or UpdateAndWarmCampaign logs and keeps prior snapshot.
//   - Dual slot-map path: broker for promptness, PG poll as safety net when broker unavailable.
//   - Campaign watcher reconnect loop vs LISTEN/NOTIFY: broker poll chosen for appliance WAL reliability (tradeoffs.mdc).
//
// Forbidden:
//   - Synchronous broker or Postgres fetch inside FilterEngine.Check, processTrack, or Tier A gnet thread.
//
// Verify (subpackage has no *_test.go; registry and slot-map behavior tested from parent packages):
//
//	go test ./internal/ingest/ -short -run TestRegistry_StartWatch -count=1
//	go test ./internal/ingest/ -short -run TestSlotMapRepo -count=1
//	go test ./internal/filter/... -short -run TestRegistry_LockFreeReadsStress -count=1
package watchers
