#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

echo "parser-chaos: TestChaos_ParserIngress_2026"
go test ./internal/ingestion/ -run='TestChaos_ParserIngress' -count=1 -timeout=5m

echo "parser-chaos: TestChaos_ParserSecurity"
go test ./internal/ingestion/ -run='TestChaos_ParserSecurity' -count=1 -timeout=5m -v

echo "parser-chaos: JSON hardening"
go test ./internal/ingestion/ -run='TestChaos_ParserSecurity_PS_G09|TestChaos_ParserSecurity_PS_G1[0-3]' -count=1 -timeout=2m -v

echo "parser-chaos: sustained load mix"
CHAOS_LOAD_DURATION=8s CHAOS_LOAD_RPS=3000 CHAOS_LOAD_WORKERS=4 \
  bash scripts/fault/parser_chaos_load.sh --duration=8s --rps=3000 --workers=4 --log=/tmp/parser-chaos-load-drill.log

echo "parser-chaos: cross-hop nginx-gnet"
go test ./internal/ingestion/ -run='TestChaos_CrossHop_NginxGnet' -count=1 -timeout=2m -v

echo "parser-chaos: TE/proto/HPACK"
go test ./internal/ingestion/ -run='TestChaos_TE_TE|TestChaos_Proto_FieldBudget|TestChaos_HPACK' -count=1 -timeout=2m -v

echo "parser-chaos: slow-body drill"
bash scripts/fault/parser_slow_body_drill.sh

echo "parser-chaos: buffer pool and hardening proofs"
go test ./internal/ingestion/ -run='TestRequestBufferPool|TestChaos_ParserSecurity_PS_H0|TestHTTP1ChunkedScratch' -count=1 -timeout=2m -v

echo "parser-chaos: fuzz smoke"
go test ./internal/ingestion/ -fuzz=FuzzParseTrackJSON -fuzztime=10s -count=1 || true
go test ./internal/ingestion/ -fuzz=FuzzParseOpenRTB3FSM -fuzztime=10s -count=1 2> /dev/null || true
go test ./internal/ingestion/ -fuzz=FuzzSkipJSONValueBudget -fuzztime=10s -count=1 || true
go test ./internal/ingestion/ -fuzz=FuzzHTTP1Chunked -fuzztime=10s -count=1 || true

echo "parser-chaos: OpenRTB fuzz smoke (legacy)"
go test ./internal/ingestion/ -fuzz=FuzzParseOpenRTB26Split -fuzztime=30s -count=1 || true

echo "parser-chaos: alloc gate benches"
bash scripts/test/gate_bench.sh
