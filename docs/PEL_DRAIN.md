# PEL drain checklist (broker cutover)

Operator one-pager for draining Redis Stream **Pending Entries List (PEL)** on `_ch` / `_pg` consumer groups before switching to **broker-only** ingest (`CH_INGEST_SOURCE=broker`).

Full migration context: [DEVELOPMENT.md §7 — Broker cutover](DEVELOPMENT.md#broker-cutover-ch_ingest_source).

---

## When to use

You are on **dual-path** (`CH_INGEST_SOURCE` empty, broker writing CH in parallel or shadow mode) and need to cut over without losing analytics or settlement events stuck in PEL.

**Do not** set `CH_INGEST_SOURCE=broker` until `_ch` PEL lag is **0** (or explicitly accepted by ops).

---

## Preconditions

| Check | Command / signal |
| :--- | :--- |
| Broker healthy | `curl -sf http://127.0.0.1:8084/health` |
| Shadow divergence clear | Prometheus `ad_broker_ingest_divergence_high` = 0 (if `BROKER_SHADOW_MODE=1`) |
| Processor up | `curl -sf http://127.0.0.1:8186/health` (port may vary) |
| Redis shards reachable | `redis-cli -a $REDIS_PASSWORD PING` on shards 0–3 |

---

## 1. Identify lagging consumer groups

Per shard (replace `redis-N`, password, stream names from your env):

```bash
# CH ingest stream (example shard 0)
redis-cli -a "$REDIS_PASSWORD" -p 6479 XPENDING ad:events:ch:0 processor-ch-group

# Settlement stream (example)
redis-cli -a "$REDIS_PASSWORD" -p 6479 XPENDING ad:events:pg:0 processor-pg-group
```

| Field | Meaning |
| :--- | :--- |
| `count` | Messages delivered but not `XACK` — **PEL size** |
| `min` / `max` | Oldest / newest pending ID |
| `consumers` | Active consumer names with pending work |

Repeat for shards **0–3** (production hot path). Shards 4–5 if used for fraud/aux streams.

Prometheus (if enabled): `ad_fraud_stream_pel_age_seconds` — alert if idle pending age grows during cutover window.

---

## 2. Drain (normal path)

1. Set `BROKER_SHADOW_MODE=0` (broker writes CH for real).
2. Keep `CH_INGEST_SOURCE=` **empty** — Redis `_ch` consumer still runs.
3. Ensure processor workers are not throttled (`PROCESSOR_WEIGHT_ENABLED`, CPU limits).
4. Poll every **30s** until all shards show `XPENDING ... count = 0`:

```bash
for i in 0 1 2 3; do
  port=$((6479 + i))
  echo "shard $i CH PEL:"
  redis-cli -a "$REDIS_PASSWORD" -p "$port" XPENDING "ad:events:ch:${i}" processor-ch-group
done
```

**Target:** `count = 0` on every shard for `_ch` (and `_pg` if still on Redis settlement).

---

## 3. Cutover

When PEL is drained:

1. Set `CH_INGEST_SOURCE=broker` in `.env` (tracker + processor).
2. `docker compose up -d --force-recreate tracker-0 tracker-1 processor`
3. Confirm logs: `Redis _ch StreamConsumer disabled (CH_INGEST_SOURCE=broker)`.
4. Verify CH row growth via broker path (`ad_broker_produced_events_total`, CH `impressions` count).

---

## 4. Rollback

1. Unset `CH_INGEST_SOURCE` (empty).
2. Restart tracker + processor — Redis `_ch` / `_pg` consumers resume.
3. Broker offsets remain on disk under `LOGGER_DIR/offsets`; no need to replay unless ops intentionally reset offsets.

---

## 5. Stuck PEL (escalation)

| Symptom | Action |
| :--- | :--- |
| `count` flat > 0, CH insert errors | Fix ClickHouse / spool; check `ad_ch_spool_*` metrics |
| Poison message / DLQ | Inspect processor logs; `XCLAIM` + DLQ stream per `internal/ingestion/processor.go` |
| Consumer group missing | Restart processor (recreates group); **do not** `XGROUP DESTROY` in prod without runbook |
| Forced cutover with lag | Accept gap in CH for pending IDs; document incident; optional `broker replay` for WAL gap only |

---

## 6. Post-cutover RAM check

After broker-only + aggressive `XTRIM`, run RAM proof and optional dual baseline compare:

```bash
bash scripts/perf/redis_ram_proof.sh
bash scripts/perf/redis_ram_cutover_compare.sh   # dual-path vs broker-only peak
```

See [BENCHMARKS.md §C](BENCHMARKS.md#c-redis-ram-proof-milestone-phase-3).
