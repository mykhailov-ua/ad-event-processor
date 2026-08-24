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

if rg -q 't\.Skip\(\)\s*$' "${SCOPE[@]}" pkg/ --glob '*_test.go' 2> /dev/null; then
  fail "bare t.Skip() without reason"
  rg 't\.Skip\(\)\s*$' "${SCOPE[@]}" pkg/ --glob '*_test.go' || true
fi

bad_skip_patterns=(
  't\.Skip\("skipping integration test'
  't\.Skip\("integration test"\)'
  't\.Skip\("skipping [^"]*in short mode"\)'
  't\.Skip\("requires postgres"\)'
  't\.Skip\("redis integration"\)'
)
for pat in "${bad_skip_patterns[@]}"; do
  if rg -q "$pat" "${SCOPE[@]}" --glob '*_test.go' 2> /dev/null; then
    fail "weak skip reason (use integration: prefix) — pattern $pat"
    rg -n "$pat" "${SCOPE[@]}" --glob '*_test.go' || true
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
done < <(rg -l 'if testing\.Short\(\)' "${SCOPE[@]}" --glob '*_test.go' 2> /dev/null || true)

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

while IFS= read -r dir; do
  [[ -z "$dir" ]] && continue
  if rg -q 'testcontainers' "$dir" --glob '*_test.go' 2> /dev/null \
    && rg -l 'httptest\.New(Server|UnstartedServer)' "$dir" --glob '*_test.go' 2> /dev/null \
    | rg -q .; then
    if rg -n 'httptest\.New(Server|UnstartedServer)' "$dir" --glob '*_test.go' 2> /dev/null \
      | rg -v 'integration:|// mock HTTP upstream|// not a database' > /dev/null; then
      fail "httptest server in package with testcontainers ($dir) — use real PG/Redis for transaction tests"
      rg -n 'httptest\.New(Server|UnstartedServer)' "$dir" --glob '*_test.go' || true
    fi
  fi
done < <(find internal/controlplane internal/payment internal/ingestion -mindepth 1 -maxdepth 1 -type d 2> /dev/null)

if rg -n 'nolint:errcheck' internal/controlplane internal/ingestion internal/payment pkg/ \
  --glob '*.go' --glob '!*_test.go' 2> /dev/null \
  | rg -v 'nolint:errcheck.*(TODO|#[0-9]+|JIRA|GH-)' > /dev/null 2>&1; then
  warn "nolint:errcheck without ticket — prefer handling the error"
  rg -n 'nolint:errcheck' internal/controlplane internal/ingestion internal/payment pkg/ \
    --glob '*.go' --glob '!*_test.go' \
    | rg -v 'nolint:errcheck.*(TODO|#[0-9]+|JIRA|GH-)' || true
fi

if [[ "$failed" -ne 0 ]]; then
  echo "anti-slop: remediation — .cursor/rules/ci.mdc and anti-slop.mdc; integration skips: integration: run make test-integration (Docker testcontainers)"
  exit 1
fi

log_ok "anti-slop checks (scope: controlplane ingestion payment)"
