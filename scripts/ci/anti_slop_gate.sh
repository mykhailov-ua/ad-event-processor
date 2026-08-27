#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

failed=0

fail() {
  echo "anti-slop: FAIL $*"
  failed=1
}

warn() {
  echo "anti-slop: WARN $*"
}

log_ok() {
  echo "anti-slop: OK   $*"
}

SCOPE=(
  internal/controlplane
  internal/ingestion
  internal/payment
)

SKIP_SCOPE=(
  internal/controlplane
  internal/ingestion
  internal/payment
  internal/fraud
  internal/rtb
  internal/licensing
  internal/identity
  internal/notify
  internal/costsync
  internal/edge
  internal/database
  internal/logpipeline
  pkg/
  cmd/
  tests/
)

if rg -q 't\.Skip\(\)\s*$' "${SKIP_SCOPE[@]}" --glob '*_test.go' 2> /dev/null; then
  fail "bare t.Skip() without reason"
  rg 't\.Skip\(\)\s*$' "${SKIP_SCOPE[@]}" --glob '*_test.go' || true
fi

bad_skip_patterns=(
  't\.Skip\("skipping integration test'
  't\.Skip\("integration test"\)'
  't\.Skip\("skipping [^"]*in short mode"\)'
  't\.Skip\("requires postgres"\)'
  't\.Skip\("redis integration"\)'
  't\.Skip\("fault integration test"\)'
  't\.Skip\("clickhouse integration test"\)'
)
for pat in "${bad_skip_patterns[@]}"; do
  if rg -q "$pat" "${SKIP_SCOPE[@]}" --glob '*_test.go' 2> /dev/null; then
    fail "weak skip reason (use integration: prefix) - pattern $pat"
    rg -n "$pat" "${SKIP_SCOPE[@]}" --glob '*_test.go' || true
  fi
done

short_bad=0
while IFS= read -r file; do
  [[ -z "$file" ]] && continue
  if ! awk '
    /if testing\.Short\(\)/ { want=1; next }
    want && /t\.Skip\(/ {
      if ($0 !~ /integration:/) {
        print FILENAME ":" NR ":" $0
        exit 1
      }
      want=0
      next
    }
    want && /{/ { next }
    want && !/^\s/ { want=0 }
  ' "$file"; then
    short_bad=1
  fi
done < <(rg -l 'if testing\.Short\(\)' "${SKIP_SCOPE[@]}" --glob '*_test.go' 2> /dev/null || true)

if [[ "$short_bad" -ne 0 ]]; then
  fail "testing.Short() skip must use integration: prefix (scoped packages)"
fi

if rg -n '^func BenchmarkUnifiedFilter_Check\(b \*testing\.B\)' internal/ingestion --glob '*_test.go' 2> /dev/null; then
  fail "rename BenchmarkUnifiedFilter_Check to BenchmarkUnifiedFilter_Check_mock (mock Redis harness)"
fi

if rg -n 'testing\.Testing\(\)' internal/controlplane internal/ingestion internal/payment pkg/ \
  --glob '*.go' --glob '!*_test.go' 2> /dev/null; then
  fail "testing.Testing() in production code"
fi

if rg -n 'os\.Getenv\("CI"\)' internal/controlplane internal/ingestion internal/payment pkg/ \
  --glob '*.go' --glob '!*_test.go' 2> /dev/null; then
  fail "os.Getenv(\"CI\") branch in production code"
fi

pattern_err='_ = (json\.Unmarshal|w\.Write)'
scan_err() {
  local path="$1"
  if rg -n "$pattern_err" "$path" > /dev/null 2>&1; then
    fail "ignored error in $path"
    rg -n "$pattern_err" "$path" || true
  fi
}

while IFS= read -r -d '' file; do
  scan_err "$file"
done < <(find internal/controlplane internal/payment -name '*_handlers.go' ! -name '*_test.go' -print0 2> /dev/null || true)

while IFS= read -r -d '' file; do
  scan_err "$file"
done < <(find internal/controlplane -name 'outbox_*.go' ! -name '*_test.go' -print0 2> /dev/null || true)

httptest_testcontainers_slop() {
  local file="$1"
  awk '
    /httptest\.New(Server|UnstartedServer)/ {
      if ($0 ~ /integration:|mock HTTP upstream|not a database/) { prev = $0; next }
      if (prev ~ /mock HTTP upstream/) { prev = $0; next }
      print FILENAME ":" NR ":" $0
      bad = 1
    }
    { prev = $0 }
    END { exit bad + 0 }
  ' "$file"
}

while IFS= read -r dir; do
  [[ -z "$dir" ]] && continue
  if rg -q 'testcontainers' "$dir" --glob '*_test.go' 2> /dev/null \
    && rg -l 'httptest\.New(Server|UnstartedServer)' "$dir" --glob '*_test.go' 2> /dev/null \
    | rg -q .; then
    while IFS= read -r file; do
      [[ -z "$file" ]] && continue
      if ! httptest_testcontainers_slop "$file"; then
        fail "httptest server in package with testcontainers ($dir) - annotate mock HTTP upstream on prior line or use testcontainers"
      fi
    done < <(rg -l 'httptest\.New(Server|UnstartedServer)' "$dir" --glob '*_test.go' 2> /dev/null || true)
  fi
done < <(find internal/controlplane internal/payment internal/ingestion -mindepth 1 -maxdepth 1 -type d 2> /dev/null)

httptest_pkg_roots=(
  internal/controlplane
  internal/ingestion
  internal/payment
)
for pkgdir in "${httptest_pkg_roots[@]}"; do
  if rg -q 'testcontainers' "$pkgdir" --glob '*_test.go' 2> /dev/null \
    && rg -l 'httptest\.New(Server|UnstartedServer)' "$pkgdir" --glob '*_test.go' --max-depth 1 2> /dev/null \
    | rg -q .; then
    while IFS= read -r file; do
      [[ -z "$file" ]] && continue
      if ! httptest_testcontainers_slop "$file"; then
        fail "httptest server in package root with testcontainers ($pkgdir) - annotate mock HTTP upstream on prior line or use testcontainers"
      fi
    done < <(rg -l 'httptest\.New(Server|UnstartedServer)' "$pkgdir" --glob '*_test.go' --max-depth 1 2> /dev/null || true)
  fi
done

if rg -n 'nolint:errcheck' internal/controlplane internal/ingestion internal/payment pkg/ \
  --glob '*.go' --glob '!*_test.go' 2> /dev/null \
  | rg -v 'nolint:errcheck.*(TODO|#[0-9]+|JIRA|GH-)' > /dev/null 2>&1; then
  warn "nolint:errcheck without ticket - prefer handling the error"
  rg -n 'nolint:errcheck' internal/controlplane internal/ingestion internal/payment pkg/ \
    --glob '*.go' --glob '!*_test.go' \
    | rg -v 'nolint:errcheck.*(TODO|#[0-9]+|JIRA|GH-)' || true
fi

if find internal -name 'n1_fix_bench_test.go' -print -quit 2> /dev/null | rg -q .; then
  fail "n1_fix_bench_test.go proof debris - use query_budget_test or domain holdout tests"
  find internal -name 'n1_fix_bench_test.go' 2> /dev/null || true
fi

if rg -n 'func TestN1Fix_' internal --glob '*_test.go' 2> /dev/null; then
  fail "TestN1Fix_* naming - rename to domain behavior (e.g. TestReconciliationWorker_*)"
fi

if rg -n '(echo|log) "[=]{2,}' scripts/ 2> /dev/null \
  | rg -v 'cold_path_db_in_loop_allowlist' > /dev/null 2>&1; then
  fail "script banner echo/log (== or ---) - use plain one-line messages"
  rg -n '(echo|log) "[=]{2,}' scripts/ \
    | rg -v 'cold_path_db_in_loop_allowlist' || true
fi

if rg -n 'func (Benchmark[^ (]*_Legacy|func parse[A-Za-z]*Legacy)' internal --glob '*_test.go' 2> /dev/null; then
  fail "Legacy proof-debris benchmarks or parse*Legacy helpers in tests"
fi

if rg -n 'func Test[A-Za-z0-9]+_m[0-9]+\(' internal --glob '*_test.go' 2> /dev/null; then
  fail "milestone trash test names (Test*_mN) - use domain behavior names"
fi

if rg -n '[\u2013\u2014\u2026\u00b7]' scripts/ --glob '*.sh' 2> /dev/null \
  | rg -v 'cold_path_db_in_loop_allowlist' > /dev/null 2>&1; then
  fail "Unicode dash or ellipsis in scripts/*.sh echo/log (use ASCII - and ...)"
  rg -n '[\u2013\u2014\u2026\u00b7]' scripts/ --glob '*.sh' \
    | rg -v 'cold_path_db_in_loop_allowlist' || true
fi

if rg -n 'for i := 0; i < b\.N' internal pkg cmd --glob '*_test.go' 2> /dev/null; then
  fail "benchmark loops must use for b.Loop() (quality.mdc)"
  rg -n 'for i := 0; i < b\.N' internal pkg cmd --glob '*_test.go' || true
fi

if rg -n '[\u2013\u2014\u2026\u00a7\u00b7]' deploy/monitoring --glob '*.yaml' 2> /dev/null; then
  fail "Unicode punctuation in deploy/monitoring alerts (use ASCII - and section)"
  rg -n '[\u2013\u2014\u2026\u00a7\u00b7]' deploy/monitoring --glob '*.yaml' || true
fi

if [[ "$failed" -ne 0 ]]; then
  echo "anti-slop: remediation - .cursor/rules/ci.mdc and anti-slop.mdc; integration skips: integration: run make test-integration (Docker testcontainers)"
  exit 1
fi

log_ok "anti-slop checks (scope: controlplane ingestion payment fraud rtb pkg tests)"
