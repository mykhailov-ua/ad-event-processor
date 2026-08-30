// Package regionproxy implements the enterprise multi-region gnet ingress proxy and WAL uplink server.
//
// Role:
//   - cmd/region-proxy binary wires Server (server.go, conn_gnet.go): gnet listener, mmap WAL, broker coordinator.
//   - ingest_batch.go: IngestBatch dedupes regional spend batches; global control uses leased book/execute path.
//   - ingest_decode.go: JSON batch wire decode (region_code, source_epoch, seq, factor_u, payload).
//   - pkg/regionproxy/{wal,uplink,keygen,opkey} for persistence and optional GLOBAL_INGEST uplink.
//
// Topology:
//   - Satellite regions forward deduped batches to primary region-proxy; primary appends WAL and broker topic
//     DefaultIngressTopic (region-proxy-ingress).
//   - Global control IngestBatch uses internal/dedup adapter with routing epoch from PG when source_epoch is zero.
//   - iogate.DiskWriteGate backs pressure-aware disk writes; health/metrics HTTP on separate listen addr.
//
// Defaults and limits:
//   - DefaultIngressTopic: region-proxy-ingress.
//   - proxyBackpressure wire status byte 7 when disk gate saturated (protocol produce-batch response).
//   - Listen addr, data dir, redis URL, and uplink knobs from cmd/region-proxy flags (see cmd/region-proxy/doc.go).
//
// Invariants:
//   - factor_u must match dedupkey.FactorU over canonical (seq, payload); mismatch rejects whole batch.
//   - dedup.GuardOutcome rejects epoch mismatch and duplicate claims without partial apply.
//   - Disk gate shedding returns proxyBackpressure, not silent drop.
//   - Read-idle and max-lifetime close partial gnet frames (conn_idle_test.go).
//
// Forbidden:
//   - Postgres settlement on gnet ingress thread.
//   - internal/ingest hot path importing pkg/regionproxy (regions.mdc).
//   - Active on default single_vps profile without multi_region license JWT feature.
//
// Verify:
//
//	go test ./internal/regionproxy/... -short -count=1
//	go test ./internal/regionproxy/ -short -run TestRegionProxy_BackpressureWhenDegraded -count=1
//	go test ./internal/regionproxy/ -short -run TestRegionProxy_ProduceBatchIngress -count=1
//	go test ./internal/regionproxy/ -short -run TestSlowClient -count=1
package regionproxy
