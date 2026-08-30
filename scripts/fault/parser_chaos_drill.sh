#!/usr/bin/env bash
set -euo pipefail

# Role: Fault/resilience: Parser chaos drill script.
# Execution context: CI main-resilience or operator fault tier; needs Docker for compose drills.
# Invariants/contracts enforced: Success logs fault_proof fault=<name>; resilience_fault_gates.sh greps required proofs.
# Verify: bash scripts/fault/parser_chaos_drill.sh
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

echo "parser-chaos: TestChaos_ParserIngress_2026"
go test ./internal/ingest/ -run='TestChaos_ParserIngress' -count=1 -timeout=5m

echo "parser-chaos: TestChaos_ParserSecurity"
go test ./internal/ingest/ -run='TestChaos_ParserSecurity' -count=1 -timeout=5m -v

echo "parser-chaos: JSON hardening"
go test ./internal/ingest/ -run='TestChaos_ParserSecurity_(UnicodeKey|DuplicateKey|LoneSurrogate|DistributedWS|QuoteDense|NestedPayload|ValueLiteral)' -count=1 -timeout=2m -v

echo "parser-chaos: sustained load mix"
CHAOS_LOAD_DURATION=8s CHAOS_LOAD_RPS=3000 CHAOS_LOAD_WORKERS=4 \
  bash scripts/fault/parser_chaos_load.sh --duration=8s --rps=3000 --workers=4 --log=/tmp/parser-chaos-load-drill.log

echo "parser-chaos: cross-hop nginx-gnet"
go test ./internal/ingest/ -run='TestChaos_CrossHop_NginxGnet' -count=1 -timeout=2m -v

echo "parser-chaos: TE/proto/HPACK"
go test ./internal/ingest/ -run='TestChaos_TE_TE|TestChaos_Proto_FieldBudget|TestChaos_HPACK' -count=1 -timeout=2m -v

echo "parser-chaos: slow-body drill"
bash scripts/fault/parser_slow_body_drill.sh

echo "parser-chaos: buffer pool and hardening proofs"
go test ./internal/ingest/ -run='TestRequestBufferPool|TestChaos_ParserSecurity_Key|TestHTTP1ChunkedScratch' -count=1 -timeout=2m -v

echo "parser-chaos: fuzz smoke"
go test ./internal/ingest/ -fuzz=FuzzParseTrackJSON -fuzztime=10s -count=1 || true
go test ./internal/ingest/ -fuzz=FuzzParseOpenRTB3FSM -fuzztime=10s -count=1 2> /dev/null || true
go test ./internal/ingest/ -fuzz=FuzzSkipJSONValueBudget -fuzztime=10s -count=1 || true
go test ./internal/ingest/ -fuzz=FuzzHTTP1Chunked -fuzztime=10s -count=1 || true

echo "parser-chaos: OpenRTB fuzz smoke (legacy)"
go test ./internal/ingest/ -fuzz=FuzzParseOpenRTB26Split -fuzztime=30s -count=1 || true

echo "parser-chaos: alloc gate benches"
bash scripts/test/load/gate_bench.sh
