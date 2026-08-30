// Package watchers runs background campaign registry and slot-map reload loops.
//
// Role:
//   - CampaignUpdateWatcher consumes broker campaign-update topic and patches filter.Registry.
//   - SlotMapWatcher reloads StaticSlot shard table from Postgres on interval.
//
// Topology:
//   - Background goroutines started from cmd/tracker wire; not on synchronous /track path.
//   - Broker consumer with backoff; optional BrokerRedisURL for offset commits.
//
// Invariants:
//   - Registry reload uses atomic.Pointer swap; hot path readers never block on watcher I/O.
//
// Forbidden:
//   - Synchronous broker or Postgres fetch inside FilterEngine.Check or processTrack.
//
// Verify:
//
//	go test ./internal/filter/... -short -run TestRegistry -count=1
//	go test ./internal/ingest/ -short -count=1
package watchers
