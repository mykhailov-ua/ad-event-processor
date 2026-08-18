#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/lint_go_paths.sh"
cd "$ROOT"

pattern='fmt\.Sprintf|context\.WithTimeout|context\.WithCancel|context\.WithValue'
fail=0

domain_forbid_skip() {
  case "$(basename "$1")" in
    broker_* | store_* | worker_* | control_* | ledger_* | settlement.go | margin_* | recon_* | migration_* | consent* | payment.go | invoice.go | customer.go | *_cold.go | campaign_update_redis.go | slot_migration_* | redis_migrate.go | pguuid.go | cohort.go | settings_rtb.go | rtb_deals_reload.go | rtb_reload.go | ctv_* | broker_*)
      return 0
      ;;
  esac
  return 1
}

echo "lint_go_hotpath_forbid_gate: scanning hot request-path sources..."

for file in "${lint_go_hot_path_request_files[@]}"; do
  [[ -f "$file" ]] || continue
  if rg -n "$pattern" "$file" > /dev/null 2>&1; then
    echo "lint_go_hotpath_forbid_gate: forbidden pattern in $file" >&2
    rg -n "$pattern" "$file" >&2 || true
    fail=1
  fi
done

while IFS= read -r -d '' file; do
  domain_forbid_skip "$file" && continue
  if rg -n "$pattern" "$file" > /dev/null 2>&1; then
    echo "lint_go_hotpath_forbid_gate: forbidden pattern in $file" >&2
    rg -n "$pattern" "$file" >&2 || true
    fail=1
  fi
done < <(find internal/domain -maxdepth 1 -type f -name '*.go' ! -name '*_test.go' -print0 2> /dev/null || true)

if [[ "$fail" -ne 0 ]]; then
  echo "lint_go_hotpath_forbid_gate: FAILED" >&2
  exit 1
fi

echo "lint_go_hotpath_forbid_gate: OK"
