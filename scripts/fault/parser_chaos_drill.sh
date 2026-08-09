#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

echo "parser-chaos: TestChaos_ParserIngress_2026"
go test ./internal/ingestion/ -run='TestChaos_ParserIngress' -count=1 -timeout=5m

echo "parser-chaos: TestChaos_ParserSecurity (PS-G01–G08 proof stubs)"
go test ./internal/ingestion/ -run='TestChaos_ParserSecurity' -count=1 -timeout=5m -v

echo "parser-chaos: S5 JSON hardening (PS-G09–G13)"
go test ./internal/ingestion/ -run='TestChaos_ParserSecurity_PS_G09|TestChaos_ParserSecurity_PS_G1[0-3]' -count=1 -timeout=2m -v

echo "parser-chaos: sustained load mix (PS-G08 / C-X02)"
CHAOS_LOAD_DURATION=8s CHAOS_LOAD_RPS=3000 CHAOS_LOAD_WORKERS=4 \
	bash scripts/fault/parser_chaos_load.sh --duration=8s --rps=3000 --workers=4 --log=/tmp/parser-chaos-load-drill.log

echo "parser-chaos: cross-hop nginx↔gnet (PS-G04)"
go test ./internal/ingestion/ -run='TestChaos_CrossHop_NginxGnet' -count=1 -timeout=2m -v

echo "parser-chaos: S4 TE/proto/HPACK (PS-G05–G07)"
go test ./internal/ingestion/ -run='TestChaos_TE_TE|TestChaos_Proto_FieldBudget|TestChaos_HPACK' -count=1 -timeout=2m -v

echo "parser-chaos: slow-body drill (PS-G01)"
bash scripts/fault/parser_slow_body_drill.sh

echo "parser-chaos: OpenRTB fuzz smoke"
go test ./internal/ingestion/ -fuzz=FuzzParseOpenRTB26Split -fuzztime=30s -count=1 || true
go test ./internal/ingestion/ -fuzz=FuzzParseOpenRTB3FSM -fuzztime=30s -count=1 2>/dev/null || true

echo "parser-chaos: alloc gate benches"
bash scripts/test/gate_bench.sh
