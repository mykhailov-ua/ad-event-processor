// Package filterwire wires FilterEngine construction, protobuf eval wire, and ingest-side filter glue.
//
// Role:
//   - Type aliases and constructors bridging internal/filter and internal/filter/unified into ingest handlers.
//   - Proto field-budget wire decode for filter eval paths; re-exports filter sentinel errors.
//   - BrandCreativeStore (atomic snapshot + bounded Redis reload) and SegmentConversionHandler (cold conversion path).
//   - Redis Lua observer helpers for per-shard metrics (proto_wire.go).
//
// Topology:
//   - FilterEngine.Check runs on caller goroutine (tracker: PinnedWorkerPool Tier B, sync, not detached).
//   - UnifiedFilter last in chain; at most one EVALSHA per accept when not local-quanta full-skip.
//   - Filter logic and Lua scripts live in internal/filter; this package is wiring and ingest-adjacent helpers only.
//
// Invariants:
//   - UnmarshalAdEventVT scans protobuf wire field count before vtproto decode (protoWireFieldCount).
//   - BrandCreativeStore LoadFromRedis uses filter deadline remaining on evt when present.
//   - SegmentConversionHandler runs only for ConversionEventType with non-empty UserID (not on /track hot accept).
//   - No Postgres or ClickHouse from FilterEngine.Check wiring paths in this package.
//
// Contracts:
//
// Env knobs:
//   - PROTO_MAX_FIELDS (default 256): ConfigureProtoMaxFields; zero or negative resets to ProtoMaxFields const.
//   - FILTER_TIMEOUT_MS (ms): passed to NewFilterEngine and BrandCreativeStore redis load timeout.
//   - JSON_STRICT_UTF8: not read here; parser.ConfigureSecurity owns UTF-8 policy.
//
// Defaults and limits:
//   - ProtoMaxFields = 256 (proto_wire.go); ErrProtoFieldBudget when wire exceeds budget.
//   - BrandCreativeStore redis load: FilterRedisReadTimeoutMs from FILTER_TIMEOUT_MS when set.
//   - SegmentConversionHandler: 3 s background timeout for conversion segment upsert (not request-path).
//   - Tracker production filter order wired in cmd/tracker/wire.go (license through unified).
//
// Tradeoffs:
//   - filterwire package vs inlining filter imports in ingest handler bundles:
//     Centralizes type aliases, sentinel errors, and ingest-only glue (BrandCreativeStore, segment handler,
//     proto budget) so handler bundles depend on one stable surface. Rejected duplicating filter types in
//     ingest: divergent error sentinels and constructor drift across bundles.
//   - Wiring-only boundary vs moving FilterEngine into ingest:
//     Check, Lua, and local filter implementations stay in internal/filter for reuse and static gates.
//     filterwire re-exports constructors; ingest calls NewFilterEngine from here, not filter internals scattered
//     across track_bundle and landing_bundle.
//   - Proto field-budget pre-scan vs blind UnmarshalVT:
//     protoWireFieldCount rejects field floods before vtproto alloc. Rejected decode-then-validate: protobuf
//     bombs would allocate past hot-path budgets (TestChaos_ParserSecurity protobuf cases).
//   - BrandCreativeStore atomic snapshot vs per-request Redis GET:
//     Snapshot served from atomic.Value; miss triggers bounded LoadFromRedis under filter deadline.
//     Rejected sync GET on every /click: adds Redis RTT on hot path.
//   - json.Unmarshal on brand creative replica only vs on track accept:
//     Cold replica parse in LoadFromRedis is off accept path; track JSON uses parser scan budgets.
//   - SegmentConversionHandler async PG/Redis vs inline on accept:
//     Conversion events enqueue segment membership in background; keeps /track accept off Postgres.
//
// Forbidden:
//   - Postgres or ClickHouse queries inside synchronous Check from this wiring layer.
//   - internal/fraud ML scoring import; boost snapshot only via filter package.
//
// Verify:
//
//	go test ./internal/ingest/ -short -run TestFilterEngine -count=1
//	go test ./internal/filter/... -short -count=1
package filterwire
