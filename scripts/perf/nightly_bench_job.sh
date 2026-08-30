#!/usr/bin/env bash
# Role: Nightly bench job comparing redis_lua or broker benches against .ci-baselines via perf-gate.
# Execution context: CI self-hosted runner; KIND arg redis|broker selects bench pattern and package.
# Env knobs: none (baseline under .ci-baselines/<kind>/).
# Verify: bash scripts/perf/nightly_bench_job.sh redis
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

KIND="${1:-}"
case "$KIND" in
  redis)
    BASELINE_DIR=".ci-baselines/redis"
    BENCH_PATTERN='BenchmarkUnifiedFilter_Check_RealRedis'
    BENCH_PKG='./internal/ingest'
    OUT_BENCH="redis_lua_bench.txt"
    OUT_GATE="redis_lua_gate.txt"
    RUN_SQLC=1
    ;;
  broker)
    BASELINE_DIR=".ci-baselines/broker"
    BENCH_PATTERN='Benchmark(BrokerThroughput|SegmentWrite)'
    BENCH_PKG='./internal/broker/... ./pkg/broker/log/... ./pkg/broker/protocol/'
    OUT_BENCH="broker_proto_bench.txt"
    OUT_GATE="broker_proto_gate.txt"
    RUN_SQLC=0
    ;;
  *)
    echo "usage: $0 redis|broker" >&2
    exit 2
    ;;
esac

mkdir -p "$BASELINE_DIR"
"$SCRIPTS/perf/install_benchstat.sh"

if [[ "$RUN_SQLC" -eq 1 ]]; then
  safe_validate_sqlc_yml "$ROOT/sqlc.yaml"
  go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.28.0 generate
fi

"$SCRIPTS/perf/run_bench.sh" "$BENCH_PATTERN" "$BENCH_PKG" | tee "$OUT_BENCH"
"$SCRIPTS/perf/perf_baseline_gate.sh" "$BASELINE_DIR/bench.txt" "$OUT_BENCH" | tee "$OUT_GATE"
cp "$OUT_BENCH" "$BASELINE_DIR/bench.txt"
